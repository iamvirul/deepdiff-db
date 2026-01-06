class DeepDiffDb < Formula
  desc "High-performance CLI tool for comparing databases and generating safe migrations"
  homepage "https://github.com/iamvirul/deepdiff-db"
  url "https://github.com/iamvirul/deepdiff-db/archive/refs/tags/v0.6.tar.gz"
  sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
  license ""
  head "https://github.com/iamvirul/deepdiff-db.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/deepdiffdb"
  end

  test do
    # Test that the binary runs and shows version/help
    assert_match "deepdiffdb", shell_output("#{bin}/deepdiffdb --help")
  end
end
