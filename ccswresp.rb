# Homebrew Formula for ccswresp
# Usage:
#   brew tap uhozicloud/ccswresp
#   brew install ccswresp
#   brew services start ccswresp

class Ccswresp < Formula
  desc "Protocol translation proxy: OpenAI Responses API ↔ Chat Completions API"
  homepage "https://github.com/uhozicloud/ccswresp"
  license "MIT"
  version "1.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/uhozicloud/ccswresp/releases/download/v1.0.0/ccswresp_darwin_arm64.tar.gz"
      sha256 "6ca1967a2621a6a9d34d8757790870a20501507f2126dc864d6220d2b64fb6f0"
    else
      url "https://github.com/uhozicloud/ccswresp/releases/download/v1.0.0/ccswresp_darwin_amd64.tar.gz"
      sha256 "6b08dcf9132a525ec5c5dcbfa3ccbd61885af295de0845017d859770020588b9"
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
    system bin/"ccswresp", "--version"
  end
end
