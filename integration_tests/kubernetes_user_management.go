package integration_tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type ClusterUsers []*ClusterUser

type ClusterUser struct {
	Name                string
	ARN                 string
	KubectlUserName     string
	Base64EncodedCSR    string
	KeyFilePath         string
	CSRFilePath         string
	CSRManifestPath     string
	CertificateFilePath string
}

var (
	// JobRequesterUser is the user to use for creating JobRequest resources
	JobRequesterUser = &ClusterUser{
		Name: "job-requester",
		ARN:  "arn:aws:sts::123456789012:assumed-role/job.req-developer/e2e",
	}
	// JobReviewerUser is the user to use for creating JobRequestReview resources
	JobReviewerUser = &ClusterUser{
		Name: "job-reviewer",
		ARN:  "arn:aws:sts::123456789012:assumed-role/job.rev-developer/e2e",
	}
	// KubernetesUsers will have kubernetes users provisioned into the cluster. Only Name and ARN need to be specified
	KubernetesUsers = &ClusterUsers{
		JobRequesterUser,
		JobReviewerUser,
	}
)

func SetupKubernetesUsers(ctx context.Context) error {
	tempDir, err := os.MkdirTemp("", "govuk-cli-integration-tests-*")
	if err != nil {
		return err
	}

	for _, user := range *KubernetesUsers {
		err = os.Mkdir(filepath.Join(tempDir, user.Name), 0700)
		if err != nil {
			return err
		}
	}

	for _, user := range *KubernetesUsers {
		user.KeyFilePath = filepath.Join(tempDir, user.Name, "e2e-cert.key")
		user.CSRFilePath = filepath.Join(tempDir, user.Name, "e2e-cert.csr")
		user.CSRManifestPath = filepath.Join(tempDir, user.Name, "e2e-csr-manifest.yaml")
		user.CertificateFilePath = filepath.Join(tempDir, user.Name, "e2e-signed.crt")
		user.KubectlUserName = fmt.Sprintf("govuk-job-request-operator-e2e-%s", user.Name)

		cmd := exec.CommandContext(ctx, "openssl", "genrsa", "-out", user.KeyFilePath)
		_, err = runCmd(cmd)
		if err != nil {
			return err
		}

		// The /'s in the ARNs must have an escape character in the final arg sent to the openssl command to be a valid CN
		cmd = exec.CommandContext(
			ctx, "openssl", "req", "-new",
			"-key", user.KeyFilePath, "-out", user.CSRFilePath,
			"-subj", fmt.Sprintf("/CN=%s", strings.ReplaceAll(user.ARN, "/", "\\/")),
		)
		_, err = runCmd(cmd)
		if err != nil {
			return err
		}

		csr, err := os.ReadFile(user.CSRFilePath)
		if err != nil {
			return err
		}

		user.Base64EncodedCSR = base64.StdEncoding.EncodeToString(csr)

		err = renderTemplate(ctx, "user_setup/certificate_signing_request.template.yaml", user.CSRManifestPath, user)
		if err != nil {
			return err
		}

		err = applyKubernetesManifest(ctx, user.CSRManifestPath)
		if err != nil {
			return err
		}

		_, err = kubectl(ctx, "certificate", "approve", user.Name)
		if err != nil {
			return err
		}

		_, err = kubectl(ctx, "wait", "csr", user.Name, "--for", "jsonpath={.status.certificate}", "--timeout", "1m")
		if err != nil {
			return err
		}

		base64EncodedSignedCertificate, err := kubectl(ctx, "get", "csr", user.Name, "-o", "jsonpath={.status.certificate}")
		if err != nil {
			return err
		}

		signedCertificate, err := base64.StdEncoding.DecodeString(base64EncodedSignedCertificate)
		if err != nil {
			return err
		}

		err = os.WriteFile(user.CertificateFilePath, signedCertificate, 0600)
		if err != nil {
			return err
		}

		_, err = kubectl(ctx,
			"config", "set-credentials", user.KubectlUserName,
			fmt.Sprintf("--client-key=%s", user.KeyFilePath),
			fmt.Sprintf("--client-certificate=%s", user.CertificateFilePath),
			"--embed-certs=true",
		)
		if err != nil {
			return err
		}
	}

	roleBindingManifestFilePath := filepath.Join(tempDir, "role_bindings.yaml")
	err = renderTemplate(ctx, "user_setup/role_binding.template.yaml", roleBindingManifestFilePath, *KubernetesUsers)
	if err != nil {
		return err
	}

	err = applyKubernetesManifest(ctx, roleBindingManifestFilePath)
	if err != nil {
		return err
	}

	return nil
}

func DeleteKubernetesUsersFromKubeconfig(ctx context.Context) error {
	for _, user := range *KubernetesUsers {
		cmd := exec.CommandContext(ctx, "kubectl", "config", "delete-user", user.KubectlUserName)
		_, err := runCmd(cmd)
		// This is only called in shutdown, and we don't want to fail the suite shutdown if this errors, so don't Expect success
		if err != nil {
			return fmt.Errorf("failed to delete user %s from kubectl config, error was %s", user.KubectlUserName, err.Error())
		}
	}

	return nil
}

func SwitchToKubernetesAdminUser() error {
	return switchToUser("kwok-admin")
}

func SwitchToKubernetesUser(clusterUser *ClusterUser) error {
	return switchToUser(clusterUser.KubectlUserName)
}

func switchToUser(kubectlUserName string) error {
	cmd := exec.CommandContext(context.Background(), "kubectl", "config", "set-context", "--current", "--user", kubectlUserName)
	_, err := runCmd(cmd)
	return err
}

func renderTemplate(ctx context.Context, templatePath, outputPath string, templateData any) (err error) {
	fixturePath, err := retrieveFixtureFilePath(ctx, templatePath)
	if err != nil {
		return err
	}

	parsedTemplate, err := template.ParseFiles(fixturePath)
	if err != nil {
		return err
	}

	fileWriter, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() {
		err = fileWriter.Close()
	}()

	err = parsedTemplate.Execute(fileWriter, templateData)
	if err != nil {
		return err
	}

	return nil
}

// Run executes the provided command within this context
func runCmd(cmd *exec.Cmd) (string, error) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf(
			"%q failed with error %q: %w", strings.Join(cmd.Args, " "), string(output), err,
		)
	}

	return string(output), nil
}

// getProjectDir will return the directory where the project is
func getProjectDir(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get project directory: %w", err)
	}
	return strings.Trim(string(out), "\n"), nil
}

func retrieveFixtureFilePath(ctx context.Context, fixture string) (string, error) {
	dir, err := getProjectDir(ctx)

	if err != nil {
		return dir, fmt.Errorf("failed to get current working directory: %w", err)
	}

	return filepath.Join(dir, "integration_tests", "fixtures", fixture), nil
}

func applyKubernetesManifest(ctx context.Context, manifestPath string) error {
	_, err := kubectl(ctx, "apply", "-f", manifestPath)
	return err
}
