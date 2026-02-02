package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/baighasan/kubecraft/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

type ServerInfo struct {
	Name     string
	Status   string // "running" or "stopped"
	NodePort int32
	Age      time.Time
}

func (c *Client) CheckNodeCapacity() error {
	pods, err := c.clientset.
		CoreV1().
		Pods("").
		List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: config.CommonLabelKey + "=" + config.CommonLabelValuePod,
				FieldSelector: "status.phase=Running",
			},
		)
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	var totalMemoryRequested int64
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			memoryRequested := container.Resources.Requests.Memory().Value() / 1024 / 1024
			totalMemoryRequested += memoryRequested
		}
	}

	if (config.TotalAvailableRAM - totalMemoryRequested) < config.CapacityThreshold {
		return fmt.Errorf("not enough ram available to allocate to server")
	}

	return nil
}

func (c *Client) AllocateNodePort() (int32, error) {
	services, err := c.clientset.
		CoreV1().
		Services("").
		List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: config.CommonLabelSelector,
			},
		)
	if err != nil {
		return 0, fmt.Errorf("failed to list services: %w", err)
	}

	occupiedPorts := make(map[int32]bool)
	for _, svc := range services.Items {
		for _, port := range svc.Spec.Ports {
			if port.NodePort != 0 {
				occupiedPorts[port.NodePort] = true
			}
		}
	}

	for port := int32(config.McNodePortRangeMin); port <= int32(config.McNodePortRangeMax); port++ {
		if !occupiedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found in range %d-%d", config.McNodePortRangeMin, config.McNodePortRangeMax)
}

func (c *Client) CreateServer(serverName string, username string, nodePort int32) error {
	// Define nodeport service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverName,
			Namespace: c.namespace,
			Labels: map[string]string{
				config.CommonLabelKey: config.CommonLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       config.CommonLabelValuePod,
					Port:       config.MinecraftPort,
					TargetPort: intstr.FromInt(config.MinecraftPort),
					NodePort:   nodePort,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				config.CommonLabelKey: config.CommonLabelValuePod,
				"server":              serverName,
				"user":                username,
			},
		},
	}

	// Create nodeport service
	_, err := c.clientset.
		CoreV1().
		Services(c.namespace).
		Create(
			context.TODO(),
			service,
			metav1.CreateOptions{},
		)
	if err != nil {
		return fmt.Errorf("failed to create server (nodeport service): %w", err)
	}

	// Define statefulset
	replicas := int32(1)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverName,
			Namespace: c.namespace,
			Labels: map[string]string{
				config.CommonLabelKey: config.CommonLabelValuePod,
				"server":              serverName,
				"user":                username,
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: serverName,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					config.CommonLabelKey: config.CommonLabelValuePod,
					"server":              serverName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						config.CommonLabelKey: config.CommonLabelValuePod,
						"server":              serverName,
						"user":                username,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  config.CommonLabelValuePod,
							Image: config.ServerImage,
							Env: []corev1.EnvVar{
								{
									Name:  "EULA",
									Value: "TRUE",
								},
								{
									Name:  "VERSION",
									Value: "1.21.11",
								},
								{
									Name:  "GAME_MODE",
									Value: "survival",
								},
								{
									Name:  "MAX_PLAYERS",
									Value: "5",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          config.CommonLabelValuePod,
									ContainerPort: config.MinecraftPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(config.ServerCPURequest),
									corev1.ResourceMemory: resource.MustParse(config.ServerMemoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(config.ServerCPULimit),
									corev1.ResourceMemory: resource.MustParse(config.ServerMemoryLimit),
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(int(config.MinecraftPort)),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "mc",
									MountPath: "/data",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mc",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: ptr.To(config.ServerStorageClass),
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(config.ServerStorageSize),
							},
						},
					},
				},
			},
		},
	}

	// Create statefulset
	_, err = c.clientset.
		AppsV1().
		StatefulSets(c.namespace).
		Create(
			context.TODO(),
			sts,
			metav1.CreateOptions{},
		)
	if err != nil {
		// Clean up the orphaned service
		_ = c.clientset.
			CoreV1().
			Services(c.namespace).
			Delete(
				context.TODO(),
				serverName,
				metav1.DeleteOptions{},
			)

		return fmt.Errorf("failed to create server (statefulset): %w", err)
	}

	return nil
}

func (c *Client) DeleteServer(serverName string) error {
	// Delete statefulset
	err := c.clientset.
		AppsV1().
		StatefulSets(c.namespace).
		Delete(
			context.TODO(),
			serverName,
			metav1.DeleteOptions{},
		)
	if err != nil {
		return fmt.Errorf("failed to delete server (statefulset): %w", err)
	}

	// Delete nodeport service
	err = c.clientset.
		CoreV1().
		Services(c.namespace).
		Delete(
			context.TODO(),
			serverName,
			metav1.DeleteOptions{},
		)
	if err != nil {
		return fmt.Errorf("failed to delete server (service): %w", err)
	}

	// Delete pvc
	pvcName := fmt.Sprintf("mc-%s-0", serverName)
	err = c.clientset.
		CoreV1().
		PersistentVolumeClaims(c.namespace).
		Delete(
			context.TODO(),
			pvcName,
			metav1.DeleteOptions{},
		)
	if err != nil {
		return fmt.Errorf("failed to delete pvc (service): %w", err)
	}

	return nil
}

func (c *Client) ListServers() ([]ServerInfo, error) {
	servers, err := c.clientset.
		AppsV1().
		StatefulSets(c.namespace).
		List(
			context.TODO(),
			metav1.ListOptions{},
		)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	serversInfo := make([]ServerInfo, 0, len(servers.Items))
	for _, sts := range servers.Items {
		// Get status
		var status string
		if sts.Spec.Replicas != nil && *sts.Spec.Replicas == 0 {
			status = "stopped"
		} else {
			status = "running"
		}

		// Get nodeport
		svc, err := c.clientset.
			CoreV1().
			Services(c.namespace).
			Get(
				context.TODO(),
				sts.Name,
				metav1.GetOptions{},
			)
		if err != nil {
			return nil, fmt.Errorf("failed to list servers when getting nodeport (%s): %w", sts.Name, err)
		}
		nodePort := svc.Spec.Ports[0].NodePort

		// Get age
		age := sts.CreationTimestamp.Time

		serverInfo := ServerInfo{
			Name:     sts.Name,
			Status:   status,
			NodePort: nodePort,
			Age:      age,
		}

		serversInfo = append(serversInfo, serverInfo)
	}

	return serversInfo, nil
}

func (c *Client) ScaleServer(serverName string, replicas int32) error {
	if replicas < 0 || replicas > 1 {
		return fmt.Errorf("invalid number of replicas (%d) for server (%s), must be 0 or 1", replicas, serverName)
	}

	// Get sts
	sts, err := c.clientset.
		AppsV1().
		StatefulSets(c.namespace).
		Get(
			context.TODO(),
			serverName,
			metav1.GetOptions{},
		)
	if err != nil {
		return fmt.Errorf("failed to get server (statefulset): %w", err)
	}

	// Scale sts
	sts.Spec.Replicas = &replicas

	// Apply update
	_, err = c.clientset.
		AppsV1().
		StatefulSets(c.namespace).
		Update(
			context.TODO(),
			sts,
			metav1.UpdateOptions{},
		)
	if err != nil {
		return fmt.Errorf("failed to scale server (statefulset): %w", err)
	}

	return nil
}

func (c *Client) WaitForReady(serverName string) error {

	for i := 0; i < config.MaxAttempts; i++ {
		pod, err := c.clientset.
			CoreV1().
			Pods(c.namespace).
			Get(
				context.TODO(),
				serverName+"-0",
				metav1.GetOptions{},
			)
		if err == nil {
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					return nil
				}
			}
		}

		time.Sleep(config.PollInterval)
	}

	return fmt.Errorf("timed out waiting for server (%s) to become ready", serverName)
}

func (c *Client) ServerExists(serverName string) (bool, error) {
	_, err := c.clientset.
		AppsV1().
		StatefulSets(c.namespace).
		Get(
			context.TODO(),
			serverName,
			metav1.GetOptions{},
		)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to list servers: %w", err)
	}

	return true, nil
}
