package integration_tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	jrv1 "github.com/alphagov/govuk-job-request-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testEnv        *envtest.Environment
	kubeconfigPath string
	dynamicClient  *dynamic.DynamicClient
)

// operatorCRDPath resolves the CRD manifests shipped inside the
// govuk-job-request-operator module, so the CRDs installed into envtest always
// match the API version pinned in go.mod.
func operatorCRDPath() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/alphagov/govuk-job-request-operator").Output()
	if err != nil {
		return "", fmt.Errorf("resolving operator module dir: %w", err)
	}
	return filepath.Join(strings.TrimSpace(string(out)), "config", "crd", "bases"), nil
}

// startTestEnv starts an envtest API server with the JobRequest CRDs installed
// and writes a kubeconfig for it that the CLI under test can be pointed at.
func startTestEnv() error {
	crdPath, err := operatorCRDPath()
	if err != nil {
		return err
	}

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{crdPath},
		ErrorIfCRDPathMissing: true,
	}

	config, err := testEnv.Start()
	if err != nil {
		return err
	}

	user, err := testEnv.AddUser(envtest.User{Name: "govuk-cli", Groups: []string{"system:masters"}}, nil)
	if err != nil {
		return err
	}

	kubeconfig, err := user.KubeConfig()
	if err != nil {
		return err
	}

	kubeconfigFile, err := os.CreateTemp("", "govuk-cli-envtest-kubeconfig-*")
	if err != nil {
		return err
	}
	defer kubeconfigFile.Close()

	if _, err := kubeconfigFile.Write(kubeconfig); err != nil {
		return err
	}
	kubeconfigPath = kubeconfigFile.Name()

	dynamicClient, err = dynamic.NewForConfig(config)
	return err
}

func stopTestEnv() error {
	if kubeconfigPath != "" {
		os.Remove(kubeconfigPath)
	}
	if testEnv == nil {
		return nil
	}
	return testEnv.Stop()
}

// createJobRequest creates the given JobRequest in envtest. The status
// subresource is enabled on the CRD, so any status on the fixture is dropped
// by the create and has to be applied with a second, status-only update.
func createJobRequest(ctx context.Context, jr *jrv1.JobRequest) error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jr)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: obj}
	u.SetAPIVersion(jrv1.SchemeGroupVersion.String())
	u.SetKind("JobRequest")

	i := dynamicClient.Resource(jrv1.SchemeGroupVersion.WithResource("jobrequests")).Namespace(jr.Namespace)
	created, err := i.Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	status, hasStatus := obj["status"]
	if !hasStatus {
		return nil
	}
	created.Object["status"] = status
	_, err = i.UpdateStatus(ctx, created, metav1.UpdateOptions{})
	return err
}

func deleteJobRequest(ctx context.Context, jr *jrv1.JobRequest) error {
	return dynamicClient.
		Resource(jrv1.SchemeGroupVersion.WithResource("jobrequests")).
		Namespace(jr.Namespace).
		Delete(ctx, jr.Name, metav1.DeleteOptions{})
}
