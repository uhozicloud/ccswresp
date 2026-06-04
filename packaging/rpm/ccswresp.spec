# RPM Spec for ccswresp
# Build with:
#   rpmbuild -ba ccswresp.spec

Name:           ccswresp
Version:        1.0.0
Release:        1%{?dist}
Summary:        Protocol translation proxy: OpenAI Responses API ↔ Chat Completions API
License:        MIT
URL:            https://github.com/uhozicloud/ccswresp
Source0:        https://github.com/uhozicloud/ccswresp/releases/download/v%{version}/%{name}_linux_amd64

BuildArch:      x86_64
Requires:       glibc >= 2.17

%description
ccswresp is a local protocol translation proxy that converts OpenAI Responses
API requests to Chat Completions API format and vice versa. It enables Codex
CLI to work with any LLM backend that provides a Chat Completions API (DeepSeek,
OpenAI, etc.).

No runtime dependencies — it's a single static Go binary.

%prep
# No prep needed — we use a pre-built binary

%build
# No build needed — Go binary is pre-compiled

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_unitdir}
install -m 755 %{SOURCE0} %{buildroot}%{_bindir}/ccswresp

# Systemd service
cat > %{buildroot}%{_unitdir}/ccswresp.service << 'SERVICEOF'
[Unit]
Description=ccswresp - Protocol Translation Proxy
Documentation=https://github.com/uhozicloud/ccswresp
After=network.target

[Service]
Type=simple
ExecStart=%{_bindir}/ccswresp
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICEOF

%files
%{_bindir}/ccswresp
%{_unitdir}/ccswresp.service

%post
%systemd_post ccswresp.service
echo ""
echo "ccswresp v%{version} installed!"
echo "Run 'ccswresp --init' to create config, then set your API key."
echo ""
echo "Manage with systemd:"
echo "  sudo systemctl start ccswresp"
echo "  sudo systemctl enable ccswresp"
echo ""

%preun
%systemd_preun ccswresp.service

%changelog
* Thu Jun 05 2026 uhozicloud <uhouzicloud@github.com> - 1.0.0-1
- Initial release
