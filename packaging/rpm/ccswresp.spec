# RPM Spec for ccswresp
# Build with:
#   rpmbuild -ba ccswresp.spec

Name:           ccswresp
Version:        1.0.0
Release:        1%{?dist}
Summary:        Protocol translation proxy: OpenAI Responses API ↔ Chat Completions API
License:        MIT
URL:            https://github.com/hoganyu/ccswresp
Source0:        https://registry.npmjs.org/%{name}/-/%{name}-%{version}.tgz

BuildArch:      noarch
Requires:       nodejs >= 18

%description
ccswresp is a local protocol translation proxy that converts OpenAI Responses
API requests to Chat Completions API format and vice versa. It enables Codex
CLI to work with any LLM backend that provides a Chat Completions API (DeepSeek,
OpenAI, etc.).

%prep
%setup -q -c %{name}-%{version}

%build
# No build step — pure JavaScript

%install
mkdir -p %{buildroot}%{_prefix}/lib/%{name}
mkdir -p %{buildroot}%{_bindir}

# Copy all source files
cp -r * %{buildroot}%{_prefix}/lib/%{name}/

# Install npm dependencies
cd %{buildroot}%{_prefix}/lib/%{name}
npm install --production

# Create wrapper script
cat > %{buildroot}%{_bindir}/ccswresp << 'EOF'
#!/bin/bash
exec node %{_prefix}/lib/ccswresp/cli.js "$@"
EOF
chmod +x %{buildroot}%{_bindir}/ccswresp

%files
%{_bindir}/ccswresp
%{_prefix}/lib/%{name}/

%post
echo ""
echo "ccswresp v%{version} installed!"
echo "Run 'ccswresp --init' to create config, then set your API key."
echo "Run 'ccswresp --help' for all options."
echo ""

%changelog
* Thu Jun 05 2026 hoganyu <hoganyu@github.com> - 1.0.0-1
- Initial release
