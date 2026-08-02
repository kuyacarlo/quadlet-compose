Name:           quadlet-compose
Version:        0.2.0
Release:        1%{?dist}
Summary:        Generate and manage systemd units from Podman Compose files
License:        MIT
URL:            https://github.com/kuyacarlo/quadlet-compose
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.22
BuildRequires:  git

# Vendored deps — no network needed during build
Provides:       bundled(golang(*))

%description
quadlet-compose converts Docker Compose files into systemd units managed by
Podman. It handles generation, installation, enable/disable, and lifecycle
operations for rootless or system-wide containers.

%prep
%autosetup -n %{name}-%{version}

%build
export GOFLAGS="-mod=vendor"
go build -o quadlet-compose -ldflags "-X main.version=%{version}" .

# Generate shell completions
./quadlet-compose completions bash > quadlet-compose.bash
./quadlet-compose completions zsh > _quadlet-compose
./quadlet-compose completions fish > quadlet-compose.fish

%install
install -Dpm 0755 quadlet-compose %{buildroot}%{_bindir}/quadlet-compose
ln -s quadlet-compose %{buildroot}%{_bindir}/complet
install -Dpm 0644 quadlet-compose.1 %{buildroot}%{_mandir}/man1/quadlet-compose.1
ln -s quadlet-compose.1 %{buildroot}%{_mandir}/man1/complet.1

# Shell completions
install -Dpm 0644 quadlet-compose.bash %{buildroot}%{_datadir}/bash-completion/completions/quadlet-compose
install -Dpm 0644 _quadlet-compose %{buildroot}%{_datadir}/zsh/site-functions/_quadlet-compose
install -Dpm 0644 quadlet-compose.fish %{buildroot}%{_datadir}/fish/vendor_completions.d/quadlet-compose.fish

%check
export GOFLAGS="-mod=vendor"
go test ./...

%files
%license LICENSE
%{_bindir}/quadlet-compose
%{_bindir}/complet
%{_mandir}/man1/quadlet-compose.1*
%{_mandir}/man1/complet.1*
%{_datadir}/bash-completion/completions/quadlet-compose
%{_datadir}/zsh/site-functions/_quadlet-compose
%{_datadir}/fish/vendor_completions.d/quadlet-compose.fish

%changelog
* Sun Aug 02 2026 kuyacarlo <kuyacarlo@users.noreply.github.com> - 0.2.0-1
- Add --in-pod flag for gen/install/enable commands

* Sun Aug 02 2026 kuyacarlo <kuyacarlo@users.noreply.github.com> - 0.1.1-1
- Add man page
- Fix changelog date

* Sun Aug 02 2026 kuyacarlo <kuyacarlo@users.noreply.github.com> - 0.1.0-1
- Initial package
