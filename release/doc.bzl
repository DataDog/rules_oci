load("@bazel_skylib//rules:diff_test.bzl", "diff_test")
load("@io_bazel_stardoc//stardoc:stardoc.bzl", "stardoc")

def stardoc_with_diff_test(
        bzl_library_target,
        out_label,
        rule_template = "@io_bazel_stardoc//stardoc:templates/markdown_tables/rule.vm"):
    """Creates a stardoc target coupled with a diff_test for a given bzl_library.
    This is helpful for minimizing boilerplate when lots of stardoc targets are to be generated.
    Args:
        bzl_library_target: the label of the bzl_library target to generate documentation for
        out_label: the label of the output MD file
        rule_template: the label or path to the Velocity rule template to use with stardoc
    """

    out_file = out_label.replace("//", "").replace(":", "/")

    # Generate MD from .bzl
    stardoc(
        name = out_file.replace("/", "_").replace(".md", "-docgen"),
        out = out_file.replace(".md", "-docgen.md"),
        input = bzl_library_target + ".bzl",
        rule_template = rule_template,
        deps = [bzl_library_target],
    )

    # Ensure that the generated MD has been updated in the local source tree
    diff_test(
        name = out_file.replace("/", "_").replace(".md", "-difftest"),
        failure_message = "Please run \"bazel run //docs:update\"",
        # Source file
        file1 = out_label,
        # Output from stardoc rule above
        file2 = out_file.replace(".md", "-docgen.md"),
    )
