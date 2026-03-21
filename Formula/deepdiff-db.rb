class DeepDiffDb < Formula
  desc "Compare databases, detect schema drift, and generate safe SQL migration packs"
  homepage "https://iamvirul.github.io/deepdiff-db/"
  url "https://github.com/iamvirul/deepdiff-db/archive/refs/tags/v0.9.tar.gz"
  sha256 "REPLACE_WITH_SHA256_ON_RELEASE"
  license "MIT"
  head "https://github.com/iamvirul/deepdiff-db.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build",
           *std_go_args(ldflags: "-s -w -X main.version=#{version}"),
           "./cmd/deepdiffdb"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/deepdiffdb --version")
    assert_match "Usage", shell_output("#{bin}/deepdiffdb --help")
  end
end
