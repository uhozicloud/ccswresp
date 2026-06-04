# Homebrew Formula for ccswresp
# Usage:
#   brew tap uhozicloud/ccswresp
#   brew install ccswresp
#   brew services start ccswresp
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
      url "https://github.com/uhozicloud/ccswresp/releases/download/v1.0.0/ccswresp_darwin_arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILD"
    else
      url "https://github.com/uhozicloud/ccswresp/releases/download/v1.0.0/ccswresp_darwin_amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILD"
    end
  end

  service do
    run [opt_bin/"ccswresp"]
    keep_alive true
    run_type :immediate
    log_path var/"log/ccswresp.log"
    error_log_path var/"log/ccswresp.log"
    working_dir Dir.home
  end

  def install
    if Hardware::CPU.arm?
      bin.install "ccswresp_darwin_arm64" => "ccswresp"
    else
      bin.install "ccswresp_darwin_amd64" => "ccswresp"
    end
  end

  def caveats
    <<~EOS
      ccswresp has been installed!

      Quick start:
        1. Create config: ccswresp --init
        2. Start:         brew services start ccswresp
        3. Status:        brew services info ccswresp
        4. Stop:          brew services stop ccswresp

      Or run in foreground: ccswresp
      For all options: ccswresp --help
    EOS
  end

  test do
    pid = spawn bin/"ccswresp", "-p", "11436"
    sleep 2
    output = shell_output("curl -s http://127.0.0.1:11436/health")
    assert_match(/"status":"ok"/, output)
  ensure
    Process.kill("TERM", pid) if pid
    Process.wait(pid) if pid
  end
end
