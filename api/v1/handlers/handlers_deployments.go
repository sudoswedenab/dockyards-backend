package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/docker/distribution/reference"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type DeploymentType int

const (
	DeploymentTypeContainer = iota
	DeploymentTypeHelm
)

func (h *handler) createDeployment(deployment *v1.Deployment) (*appsv1.Deployment, error) {
	containerPort := 80
	if deployment.Port != nil {
		containerPort = *deployment.Port
	}

	if deployment.Name == nil {
		return nil, errors.New("deployment name must not be empty")
	}

	if deployment.ContainerImage == nil {
		return nil, errors.New("deployment image must not be empty")
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

func (h *handler) createService(deployment *v1.Deployment) (*corev1.Service, error) {
	if deployment.Name == nil {
		return nil, errors.New("deployment name must not be empty")
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

func (h *handler) parseContainerImage(ref string) (string, error) {
	named, err := reference.ParseDockerRef(ref)
	if err != nil {
		return "", err
	}

	return named.Name(), nil
}

func (h *handler) PostClusterDeployments(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		h.logger.Debug("cluster empty")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var deployment v1.Deployment
	err := c.BindJSON(&deployment)
	if err != nil {
		h.logger.Error("failed to read body", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	deployment.ClusterID = clusterID

	var deploymentType DeploymentType

	if deployment.ContainerImage != nil {
		normalizedName, err := h.parseContainerImage(*deployment.ContainerImage)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "container image is not valid",
				"name":    *deployment.ContainerImage,
				"details": err.Error(),
			})
		}

		deployment.ContainerImage = &normalizedName

		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			base := path.Base(*deployment.ContainerImage)
			deployment.Name = &base
		}

		deploymentType = DeploymentTypeContainer
	}

	if deployment.HelmChart != nil {
		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			h.logger.Debug("deployment name empty, using helm chart name", "chart", *deployment.HelmChart)

			deployment.Name = deployment.HelmChart
		}

		deploymentType = DeploymentTypeHelm
	}

	details, validName := names.IsValidName(*deployment.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    deployment.Name,
			"details": details,
		})
		return
	}

	if deployment.Namespace == nil {
		h.logger.Debug("deployment namespace empty, using deployment name", "name", *deployment.Name)

		deployment.Namespace = deployment.Name
	}

	var existingDeployment v1.Deployment
	err = h.db.Take(&existingDeployment, "name = ? AND cluster_id = ?", *deployment.Name, deployment.ClusterID).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Error("error taking deployment from database", "name", *deployment.Name, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.Debug("deployment name already in-use", "name", *deployment.Name, "cluster", deployment.ClusterID)

		c.AbortWithStatus(http.StatusConflict)
		return
	}

	deployment.ID = uuid.New()

	if deploymentType == DeploymentTypeContainer {
		h.logger.Debug("deployment type is container, creating git repository")

		appsv1Deployment, err := h.createDeployment(&deployment)
		if err != nil {
			h.logger.Error("failed to create appsv1/deployment", "name", deployment.Name, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		deploymentJson, err := json.Marshal(appsv1Deployment)
		if err != nil {
			h.logger.Error("error marshalling deployment as json", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		service, err := h.createService(&deployment)
		if err != nil {
			h.logger.Error("error creating service", "name", deployment.Name, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		serviceJson, err := json.Marshal(service)
		if err != nil {
			h.logger.Error("error mashalling service as json", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		repoPath := path.Join(h.gitProjectRoot, "/v1/deployments", deployment.ID.String())

		repo, err := git.PlainInit(repoPath, false)
		if err != nil {
			h.logger.Error("error creating git repository", "path", repoPath, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		worktree, err := repo.Worktree()
		if err != nil {
			h.logger.Error("error getting default worktree", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		filename := path.Join(repoPath, "deployment.json")
		file, err := os.Create(filename)
		if err != nil {
			h.logger.Error("error creating deployment file", "filename", filename, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		_, err = file.Write(deploymentJson)
		if err != nil {
			h.logger.Error("error writing to file", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		file.Close()
		worktree.Add("deployment.json")

		filename = path.Join(repoPath, "service.json")
		file, err = os.Create(filename)
		if err != nil {
			h.logger.Error("error creating service file", "filename", filename, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		_, err = file.Write(serviceJson)
		if err != nil {
			h.logger.Error("error writing service json to file", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		file.Close()
		worktree.Add("service.json")

		commit, err := worktree.Commit("Add deployment", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "dockyards",
				Email: "git@dockyards.io",
				When:  time.Now(),
			},
		})

		if err != nil {
			h.logger.Error("error creating commit", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		h.logger.Debug("created commit", "hash", commit.String())
	}

	err = h.db.Create(&deployment).Error
	if err != nil {
		h.logger.Error("error creating deployment in database", "name", deployment.Name, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	deploymentStatus := v1.DeploymentStatus{
		ID:           uuid.New(),
		DeploymentID: deployment.ID,
		State:        util.Ptr("pending"),
		Health:       util.Ptr(v1.DeploymentStatusHealthWarning),
	}

	err = h.db.Create(&deploymentStatus).Error
	if err != nil {
		h.logger.Warn("error creating deployment status in database", "err", err)
	}

	deployment.Status = deploymentStatus

	c.JSON(http.StatusCreated, deployment)
}

func (h *handler) GetClusterDeployments(c *gin.Context) {
	clusterID := c.Param("clusterID")

	cluster, err := h.clusterService.GetCluster(clusterID)
	if err != nil {
		h.logger.Error("error getting deployment cluster from cluster service", "id", clusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	h.logger.Debug("cluster", "organization", cluster.Organization)

	var deployments []v1.Deployment
	err = h.db.Find(&deployments, "cluster_id = ?", clusterID).Error
	if err != nil {
		h.logger.Error("error finding deployments in database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, deployments)
}

func (h *handler) DeleteDeployment(c *gin.Context) {
	deploymentID := c.Param("deploymentID")
	if deploymentID == "" {
		h.logger.Debug("deployment id empty")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var deploymentStatuses []v1.DeploymentStatus
	err := h.db.Find(&deploymentStatuses, "deployment_id = ?", deploymentID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.Error("error finding deployment statuses in database", "id", deploymentID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	for _, deploymentStatus := range deploymentStatuses {
		h.logger.Error("deleting deployment status from database", "id", deploymentStatus.ID)

		err := h.db.Delete(&deploymentStatus).Error
		if err != nil {
			h.logger.Error("error deleting deployment status from database", "id", deploymentStatus.ID, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		h.logger.Debug("deleted deployment status from database", "id", deploymentStatus.ID)
	}

	var deployment v1.Deployment
	err = h.db.Take(&deployment, "id = ?", deploymentID).Error
	if err != nil {
		h.logger.Error("error taking deployment from database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleting deployment from database", "id", deployment.ID)

	err = h.db.Delete(&deployment).Error
	if err != nil {
		h.logger.Error("error deleting deployment from database", "id", deployment.ID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleted deployment from database", "id", deployment.ID)

	repoPath := path.Join(h.gitProjectRoot, "/v1/deployments", deployment.ID.String())

	h.logger.Debug("deleting repository from filesystem", "path", repoPath)

	err = os.RemoveAll(repoPath)
	if err != nil {
		h.logger.Warn("error deleting repository from filesystem", "path", repoPath, "err", err)
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("deploymentID")

	var deployment v1.Deployment
	err := h.db.Take(&deployment, "id = ?", deploymentID).Error
	if err != nil {
		h.logger.Debug("error taking deployment from database", "id", deploymentID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var deploymentStatus v1.DeploymentStatus
	err = h.db.Order("created_at desc").First(&deploymentStatus, "deployment_id = ?", deploymentID).Error
	if err != nil {
		h.logger.Warn("error taking deployment status from database", "id", deploymentID, "err", err)
	}

	deployment.Status = deploymentStatus

	c.JSON(http.StatusOK, deployment)
}
