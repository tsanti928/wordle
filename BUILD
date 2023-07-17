load("@io_bazel_rules_go//go:def.bzl", "go_binary")

cc_binary(
  name = "wordle_cc",
  deps = [
      "@cpp-httplib//:httplib",
      "@com_google_absl//absl/flags:flag",
      "@com_google_absl//absl/flags:parse",
      "@com_google_absl//absl/strings:str_format",
      "@nlohmann_json//:json",
  ],
  srcs = [
    "wordle.cc",
  ],
)

go_binary(
    name = "wordle_go",
    srcs = [
        "wordle.go",
    ],
)

go_binary(
    name = "both",
    srcs = [
        "both.go",
    ],
)

