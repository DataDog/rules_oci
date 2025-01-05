""" public rules """

load("//oci/private:oci_image.bzl", _oci_image = "oci_image")
load("//oci/private:oci_image_index.bzl", _oci_image_index = "oci_image_index")
load("//oci/private:oci_image_layer.bzl", _oci_image_layer = "oci_image_layer")
load("//oci/private:oci_push.bzl", _oci_push = "oci_push")

oci_image = _oci_image
oci_image_index = _oci_image_index
oci_image_layer = _oci_image_layer
oci_push = _oci_push
