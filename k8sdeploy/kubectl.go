package k8sdeploy

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type kubectl struct {
}

func (k *kubectl) Apply(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", path)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logrus.Debugf("running %s", strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "kubectl apply failed: %s", stderr.String())
	}

	logrus.Debugf("kubectl apply succeeded: %s", stdout.String())

	return nil
}

func (k *kubectl) Delete(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "-f", path)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logrus.Debugf("running %s", strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "kubectl delete failed: %s", stderr.String())
	}

	logrus.Debugf("kubectl delete succeeded: %s", stdout.String())

	return nil
}
