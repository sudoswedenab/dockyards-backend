package deployment

import (
	"errors"
	"path"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/containers/image/v5/docker/reference"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

var (
	ErrUnknownDeploymentType          = errors.New("unsupported deployment type")
	ErrDeploymentNameEmpty            = errors.New("deployment name must not be empty")
	ErrDeploymentImageEmpty           = errors.New("deployment image must not be empty")
	ErrDeploymentKustomizationMissing = errors.New("no kustomization.yaml file provided")
)

func createDeployment(deployment *v1.Deployment) (*appsv1.Deployment, error) {
	containerPort := 80
	if deployment.Port != nil {
		containerPort = *deployment.Port
	}

	if deployment.Name == nil {
		return nil, ErrDeploymentNameEmpty
	}

	if deployment.ContainerImage == nil {
		return nil, ErrDeploymentImageEmpty
	}

	containerPorts := []corev1.ContainerPort{
		{
			Name:          "http",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: int32(containerPort),
		},
	}

	d := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: *deployment.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": *deployment.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": *deployment.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            *deployment.Name,
							Image:           *deployment.ContainerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports:           containerPorts,
						},
					},
				},
			},
		},
	}

	return &d, nil
}

func createService(deployment *v1.Deployment) (*corev1.Service, error) {
	if deployment.Name == nil {
		return nil, ErrDeploymentNameEmpty
	}

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: *deployment.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": *deployment.Name,
			},
		},
	}

	return &service, nil
}

func AddNormalizedName(deployment *v1.Deployment) error {
	if deployment.ContainerImage != nil {
		named, err := reference.ParseNormalizedNamed(*deployment.ContainerImage)
		if err != nil {
			return err
		}

		deployment.ContainerImage = util.Ptr(named.String())

		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			base := path.Base(named.Name())
			deployment.Name = &base
		}

		deployment.Type = v1.DeploymentTypeContainerImage
	}

	if deployment.HelmChart != nil {
		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			deployment.Name = deployment.HelmChart
		}

		deployment.Type = v1.DeploymentTypeHelm
	}

	if deployment.Kustomize != nil {
		if deployment.Name == nil {
			deployment.Name = util.Ptr("")
		}

		deployment.Type = v1.DeploymentTypeKustomize
	}

	if deployment.Namespace == nil {
		deployment.Namespace = deployment.Name
	}

	deployment.Id = uuid.New()

	return nil
}

func CreateRepository(deployment *v1.Deployment, gitProjectRoot string) error {
	switch deployment.Type {
	case v1.DeploymentTypeContainerImage:
		appsv1Deployment, err := createDeployment(deployment)
		if err != nil {
			// h.logger.Error("failed to create appsv1/deployment", "name", deployment.Name, "err", err)

			return err
		}

		deploymentYAML, err := yaml.Marshal(appsv1Deployment)
		if err != nil {
			// h.logger.Error("error marshalling deployment as json", "err", err)

			return err
		}

		service, err := createService(deployment)
		if err != nil {
			// h.logger.Error("error creating service", "name", deployment.Name, "err", err)

			return err
		}

		serviceYAML, err := yaml.Marshal(service)
		if err != nil {
			// h.logger.Error("error mashalling service as json", "err", err)

			return err
		}

		repoPath := path.Join(gitProjectRoot, "/v1/deployments", deployment.Id.String())

		fs := osfs.New(repoPath)
		storage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
		initOptions := git.InitOptions{
			DefaultBranch: plumbing.Main,
		}
		mfs := memfs.New()

		repo, err := git.InitWithOptions(storage, mfs, initOptions)
		if err != nil {
			// h.logger.Error("error creating git repository", "path", repoPath, "err", err)

			return err
		}

		worktree, err := repo.Worktree()
		if err != nil {
			// h.logger.Error("error getting default worktree", "err", err)

			return err
		}

		file, err := mfs.Create("deployment.yaml")
		if err != nil {
			// h.logger.Error("error creating deployment file", "filename", filename, "err", err)

			return err
		}

		_, err = file.Write(deploymentYAML)
		if err != nil {
			// h.logger.Error("error writing to file", "err", err)

			return err
		}

		file.Close()
		worktree.Add("deployment.yaml")

		file, err = mfs.Create("service.yaml")
		if err != nil {
			// h.logger.Error("error creating service file", "filename", filename, "err", err)

			return err
		}

		_, err = file.Write(serviceYAML)
		if err != nil {
			// h.logger.Error("error writing service json to file", "err", err)

			return err
		}

		file.Close()
		worktree.Add("service.yaml")

		_, err = worktree.Commit("Add deployment", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "dockyards",
				Email: "git@dockyards.io",
				When:  time.Now(),
			},
		})

		if err != nil {
			// h.logger.Error("error creating commit", "err", err)

			return err
		}

		// h.logger.Debug("created commit", "hash", commit.String())
	case v1.DeploymentTypeKustomize:
		kustomize := *deployment.Kustomize

		_, hasKustomization := kustomize["kustomization.yaml"]
		if !hasKustomization {
			// h.logger.Error("deployment type is kustomized but no kustomization file provided")

			return ErrDeploymentKustomizationMissing
		}

		repoPath := path.Join(gitProjectRoot, "/v1/deployments", deployment.Id.String())

		fs := osfs.New(repoPath)
		storage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
		initOptions := git.InitOptions{
			DefaultBranch: plumbing.Main,
		}
		mfs := memfs.New()

		repo, err := git.InitWithOptions(storage, mfs, initOptions)
		if err != nil {
			// h.logger.Error("error creating git repository", "path", repoPath, "err", err)

			return err
		}

		worktree, err := repo.Worktree()
		if err != nil {
			// h.logger.Error("error getting default worktree", "err", err)

			return err
		}

		for filename, content := range kustomize {
			//filepath := path.Join(repoPath, filename)
			file, err := mfs.Create(filename)
			if err != nil {
				// h.logger.Error("error create kustomize file", "filepath", filepath, "err", err)

				return err
			}

			_, err = file.Write(content)
			if err != nil {
				// h.logger.Error("error writing kustomize file", "filepath", filepath, "err", err)

				return err
			}

			file.Close()

			worktree.Add(filename)
		}

		_, err = worktree.Commit("Add kustomize", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "dockyards",
				Email: "git@dockyards.io",
				When:  time.Now(),
			},
		})

		if err != nil {
			// h.logger.Error("error creating commit", "err", err)

			return err
		}

		// h.logger.Error("created git commit", "commit", commit)
	default:
		return ErrUnknownDeploymentType
	}

	return nil
}
