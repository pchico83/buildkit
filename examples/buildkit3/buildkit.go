package main

import (
	"flag"
	"os"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/util/system"
)

type buildOpt struct {
	target     string
	containerd string
	runc       string
	buildkit   string
}

func main() {
	var opt buildOpt
	flag.StringVar(&opt.target, "target", "containerd", "target (standalone, containerd)")
	flag.StringVar(&opt.containerd, "containerd", "master", "containerd version")
	flag.StringVar(&opt.runc, "runc", "v1.0.0-rc3", "runc version")
	flag.StringVar(&opt.buildkit, "buildkit", "master", "buildkit version")
	flag.Parse()

	bk := buildkit(opt)
	out := bk
	dt, err := out.Marshal()
	if err != nil {
		panic(err)
	}
	llb.WriteTo(dt, os.Stdout)
}

func goBuildBase() llb.State {
	goAlpine := llb.Image("docker.io/library/golang:1.8-alpine")
	return goAlpine.
		AddEnv("PATH", "/usr/local/go/bin:"+system.DefaultPathEnv).
		AddEnv("GOPATH", "/go").
		Run(llb.Shlex("apk add --no-cache g++ linux-headers make")).Root()
}

func goRepo(s llb.State, repo string, src llb.State) func(ro ...llb.RunOption) llb.State {
	dir := "/go/src/" + repo
	return func(ro ...llb.RunOption) llb.State {
		es := s.Dir(dir).Run(ro...)
		es.AddMount(dir, src, llb.Readonly)
		return es.AddMount("/out", llb.Scratch())
	}
}

func runc(version string) llb.State {
	repo := "github.com/opencontainers/runc"
	src := llb.Git(repo, version)
	if version == "local" {
		src = llb.Local("runc-src")
	}
	return goRepo(goBuildBase(), repo, src)(
		llb.Shlex("go build -o /out/runc ./"),
	)
}

func containerd(version string) llb.State {
	repo := "github.com/containerd/containerd"
	src := llb.Git(repo, version, llb.KeepGitDir())
	if version == "local" {
		src = llb.Local("containerd-src")
	}
	return goRepo(
		goBuildBase().
			Run(llb.Shlex("apk add --no-cache btrfs-progs-dev")).Root(),
		repo, src)(
		llb.Shlex("go build -o /out/containerd ./cmd/containerd"),
	)
}

func buildkit(opt buildOpt) llb.State {
	repo := "github.com/moby/buildkit"
	src := llb.Git(repo, "master")
	if opt.buildkit == "local" {
		src = llb.Local("buildkit-src")
	}
	run := goRepo(goBuildBase(), repo, src)

	builddStandalone := run(llb.Shlex("go build -o /out/buildd-standalone -tags standalone ./cmd/buildd"))

	builddContainerd := run(llb.Shlex("go build -o /out/buildd-containerd -tags containerd ./cmd/buildd"))

	buildctl := run(llb.Shlex("go build -o /out/buildctl ./cmd/buildctl"))

	r := llb.Scratch().With(
		copyAll(buildctl, "/"),
		copyAll(runc(opt.runc), "/"),
	)

	if opt.target == "containerd" {
		return r.With(
			copyAll(containerd(opt.containerd), "/"),
			copyAll(builddContainerd, "/"))
	}
	return r.With(copyAll(builddStandalone, "/"))
}

func copyAll(src llb.State, destPath string) llb.StateOption {
	return copyFrom(src, "/.", destPath)
}

// copyFrom has similar semantics as `COPY --from`
func copyFrom(src llb.State, srcPath, destPath string) llb.StateOption {
	return func(s llb.State) llb.State {
		return copy(src, srcPath, s, destPath)
	}
}

// copy copies files between 2 states using cp until there is no copyOp
func copy(src llb.State, srcPath string, dest llb.State, destPath string) llb.State {
	cpImage := llb.Image("docker.io/library/alpine:latest")
	cp := cpImage.Run(llb.Shlexf("cp -a /src%s /dest%s", srcPath, destPath), llb.ReadonlyRootFS)
	cp.AddMount("/src", src, llb.Readonly)
	return cp.AddMount("/dest", dest)
}
