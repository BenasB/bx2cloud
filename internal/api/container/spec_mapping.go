package container

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	imgspecs "github.com/opencontainers/image-spec/specs-go/v1"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
)

func imageSpecToRuntimeSpec(containerId uint32, rootFsDir string, img *imgspecs.ImageConfig) *runspecs.Spec {
	dnsSource := "/etc/resolv.conf"
	const systemdDnsSource = "/run/systemd/resolve/resolv.conf"
	if _, err := os.Stat(systemdDnsSource); err == nil {
		dnsSource = systemdDnsSource
	}

	user := runspecs.User{}
	userParts := strings.Split(img.User, ":")
	if len(userParts) == 2 {
		uid, uidErr := strconv.ParseUint(userParts[0], 10, 32)
		gid, gidErr := strconv.ParseUint(userParts[1], 10, 32)
		if uidErr == nil && gidErr == nil {
			user.UID = uint32(uid)
			user.GID = uint32(gid)
		} else {
			user.Username = img.User
		}
	} else {
		if uid, err := strconv.ParseUint(img.User, 10, 32); err == nil {
			user.UID = uint32(uid)
		} else {
			user.Username = img.User
		}
	}

	var args []string
	switch {
	case len(img.Entrypoint) > 0:
		args = append([]string{}, img.Entrypoint...)
		args = append(args, img.Cmd...)
	case len(img.Cmd) > 0:
		args = []string{"/bin/sh", "-c", strings.Join(img.Cmd, " ")}
	default:
		args = []string{"/bin/sh"}
	}

	env := make([]string, len(img.Env))
	copy(env, img.Env)
	pathEnvFound := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathEnvFound = true
			break
		}
	}

	if !pathEnvFound {
		env = append(env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	}

	process := runspecs.Process{
		User: user,
		Args: args,
		Env:  env,
		Cwd:  img.WorkingDir,
	}

	spec := &runspecs.Spec{
		Version: runspecs.Version,
		Root: &runspecs.Root{
			Path:     rootFsDir,
			Readonly: false,
		},
		Process: &process,
		Mounts: []runspecs.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/dev/pts",
				Type:        "devpts",
				Source:      "devpts",
				Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"},
			},
			{
				Destination: "/sys",
				Type:        "sysfs",
				Source:      "sysfs",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/dev/shm",
				Type:        "tmpfs",
				Source:      "shm",
				Options: []string{
					"nosuid",
					"noexec",
					"nodev",
					"mode=1777",
					"size=65536k",
				},
			},
			{
				Destination: "/etc/resolv.conf",
				Type:        "bind",
				Source:      dnsSource,
				Options:     []string{"rbind", "ro"},
			},
		},
		Linux: &runspecs.Linux{
			Namespaces: []runspecs.LinuxNamespace{
				{Type: runspecs.PIDNamespace},
				{Type: runspecs.IPCNamespace},
				{Type: runspecs.UTSNamespace},
				{Type: runspecs.MountNamespace},
				{Type: runspecs.NetworkNamespace},
			},
		},
		Hostname: fmt.Sprintf("container-%d", containerId),
	}

	return spec
}
