output "instance_ip" {
  description = "IP assigned to instance"
  value = oci_core_instance.kc_instance.public_ip
}

output "ssh_command" {
  description = "Command to ssh into instance"
  value = "ssh ubuntu@${oci_core_instance.kc_instance.public_ip}"
}

output "kubeconfig_command" {
  description = "SCP command to copy kubeconfig from instance"
  value = "scp ubuntu@${oci_core_instance.kc_instance.public_ip}:/etc/rancher/k3s/k3s.yaml ~/.kube/kubecraft-config"
}

output "cluster_endpoint" {
  description = "Endpoint to access kubernetes cluster"
  value = "https://${oci_core_instance.kc_instance.public_ip}:6443"
}