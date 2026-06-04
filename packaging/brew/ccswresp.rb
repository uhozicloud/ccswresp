# Homebrew Formula for ccswresp
# Usage:
#   brew tap uhozicloud/ccswresp
#   brew install ccswresp
#
# Or install directly:
#   brew install uhozicloud/ccswresp/ccswresp

class Ccswresp < Formula
  desc "Protocol translation proxy: OpenAI Responses API ↔ Chat Completions API"
  homepage "https://github.com/uhozicloud/ccswresp"
  license "MIT"
  version "1.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/uhozicloud/ccswresp/releases/download/v1.0.0/ccswresp_darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILD"
    else
      url "https://github.com/uhozicloud/ccswresp/releases/download/v1.0.0/ccswresp_darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILD"
    end
  end

  def install
    # Binary inside the tar.gz is named "ccswresp"
    bin.install "ccswresp"
  end

  def caveats
    <<~EOS
      ccswresp has been installed!

      Quick start:
        1. Create config: ccswresp --init
        2. Edit config:   nano ~/.ccswresp/.env  (set your API key)
        3. Start:         ccswresp
        4. Point Codex CLI to http://127.0.0.1:11435/v1/responses

      For all options: ccswresp --help
    EOS
  end

  test do
    # Start the server and check health endpoint
    pid = spawn bin/"ccswresp", "-p", "11436"
    sleep 2
    output = shell_output("curl -s http://127.0.0.1:11436/health")
    assert_match(/"status":"ok"/, output)
  ensure
    Process.kill("TERM", pid) if pid
    Process.wait(pid) if pid
  end
end
