resource "oci_core_security_list" "kc_security_list" {
  compartment_id = var.compartment_ocid
  vcn_id = oci_core_vcn.kc_vcn.id
  display_name = "kubecraft-security-list"

  # One ingress security_rules block per inbound rule
  ingress_security_rules {
    protocol = "6"
    source   = "${var.my_ip}/32"
    tcp_options {
      min = 22
      max = 22
    }
  }
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 6443
      max = 6443
    }
  }
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = var.node_port_min
      max = var.node_port_max
    }
  }

  # One egress_security_rules block for outbound
  egress_security_rules {
    protocol = "all"
    destination = "0.0.0.0/0"
  }
}