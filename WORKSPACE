load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

##############
# Bazel Skylib
##############
http_archive(
    name = "bazel_skylib",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.2.1/bazel-skylib-1.2.1.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.2.1/bazel-skylib-1.2.1.tar.gz",
    ],
    sha256 = "f7be3474d42aae265405a592bb7da8e171919d74c16f082a5457840f06054728",
)
load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")
bazel_skylib_workspace()

##############
# C++
##############
http_archive(
  name = "rules_cc",
  urls = ["https://github.com/bazelbuild/rules_cc/archive/2f8c04c04462ab83c545ab14c0da68c3b4c96191.zip"],
  strip_prefix = "rules_cc-2f8c04c04462ab83c545ab14c0da68c3b4c96191",
)

http_archive(
  name = "com_google_absl",
  urls = ["https://github.com/abseil/abseil-cpp/archive/e517aaf499f88383000d4ddf6b84417fbbb48791.zip"],
  strip_prefix = "abseil-cpp-e517aaf499f88383000d4ddf6b84417fbbb48791",
)

http_archive(
  name = "nlohmann_json",
  urls = ["https://github.com/nlohmann/json/archive/refs/heads/develop.zip"],
  # strip_prefix will essentially remove the inner "json-develop" directory present in develop.zip,
  # and make the BUILD file exist at root of the archive.
  strip_prefix = "json-develop",
)

http_archive(
    name = "cpp-httplib",
    urls = ["https://github.com/yhirose/cpp-httplib/archive/refs/heads/master.zip"],
    build_file = "@//:cpp-httplib.BUILD",
    strip_prefix = "cpp-httplib-master",
)

##############
# Go
##############
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "51dc53293afe317d2696d4d6433a4c33feedb7748a9e352072e2ec3c0dafd2c6",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.40.1/rules_go-v0.40.1.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.40.1/rules_go-v0.40.1.zip",
    ],
)
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
go_rules_dependencies()
go_register_toolchains(version = "1.20.5")
