# Homebrew Formula for ccswresp
# Usage:
#   brew tap hoganyu/ccswresp
#   brew install ccswresp
#
# Or install directly:
#   brew install hoganyu/ccswresp/ccswresp

class Ccswresp < Formula
  desc "Protocol translation proxy: OpenAI Responses API ↔ Chat Completions API"
  homepage "https://github.com/hoganyu/ccswresp"
  license "MIT"
  version "1.0.0"

  # When published to npm, use the tarball URL
  url "https://registry.npmjs.org/ccswresp/-/ccswresp-1.0.0.tgz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_PUBLISH"

  depends_on "node"

  def install
    # Install as a global npm package into the prefix
    system "npm", "install", "-g", "--prefix", prefix, libexec

    # Create symlinks in bin
    bin.install_symlink Dir[libexec/"bin/*"]

    # Create config directory
    (etc/"ccswresp").mkpath
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
    pid = fork { exec bin/"ccswresp", "-p", "11436" }
    sleep 2
    output = shell_output("curl -s http://127.0.0.1:11436/health")
    assert_match(/"status":"ok"/, output)
  ensure
    Process.kill("TERM", pid) if pid
    Process.wait(pid) if pid
  end
end
