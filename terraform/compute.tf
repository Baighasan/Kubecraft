resource "oci_core_instance" "kc_instance" {
  compartment_id = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domain.ad.name
  display_name = "kubecraft-instance"
  shape = "VM.Standard.A2.Flex"

  shape_config {
    ocpus = 3
    memory_in_gbs = 16
  }

  source_details {
    source_type = "image"
    source_id = data.oci_core_images.ubuntu_arm.images[0].id
    boot_volume_size_in_gbs = 100
  }

  create_vnic_details {
    subnet_id = oci_core_subnet.kc_subnet.id
    assign_public_ip = true
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data = base64encode(file("cloud-init.yaml"))
  }
}