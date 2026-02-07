resource "oci_core_vcn" "kc_vcn" {
  compartment_id  = var.compartment_ocid
  cidr_blocks     = ["10.0.0.0/16"]
  display_name    = "kubecraft-vcn"
}

resource "oci_core_internet_gateway" "kc_internet_gateway" {
  compartment_id  = var.compartment_ocid
  vcn_id          = oci_core_vcn.kc_vcn.id
  display_name    = "kubecraft_igw"
  enabled         = true
}

resource "oci_core_route_table" "kc_route_table" {
  compartment_id  = var.compartment_ocid
  vcn_id          = oci_core_vcn.kc_vcn.id
  display_name    = "kubecraft-rt"

  route_rules {
    destination = "0.0.0.0/0"
    network_entity_id = oci_core_internet_gateway.kc_internet_gateway.id
  }
}

resource "oci_core_subnet" "kc_subnet" {
  compartment_id = var.compartment_ocid
  vcn_id = oci_core_vcn.kc_vcn.id
  cidr_block = "10.0.1.0/24"
  display_name = "kubecraft-public-subnet"
  route_table_id = oci_core_route_table.kc_route_table.id
  security_list_ids = [oci_core_security_list.kc_security_list.id]
}