class Tmc < Formula
  desc "TMCopilot command-line client"
  homepage "https://github.com/huski-inc/tmcopilot-cli"
  version "0.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/huski-inc/tmcopilot-cli/releases/download/v#{version}/tmc-#{version}-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    else
      url "https://github.com/huski-inc/tmcopilot-cli/releases/download/v#{version}/tmc-#{version}-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/huski-inc/tmcopilot-cli/releases/download/v#{version}/tmc-#{version}-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    else
      url "https://github.com/huski-inc/tmcopilot-cli/releases/download/v#{version}/tmc-#{version}-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    end
  end

  def install
    bin.install "tmc"
    bin.install_symlink bin/"tmc" => "tmcopilot"
  end

  test do
    system "#{bin}/tmc", "version"
    system "#{bin}/tmcopilot", "version"
  end
end
