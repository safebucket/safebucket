<a href="https://docs.safebucket.io" target="_blank" rel="noopener">
  <picture>
    <img src="./assets/safebucket_banner.png" alt="Safebucket">
  </picture>
</a>

<div align="center">
  <h2>
    An open source file sharing platform with pluggable infrastructure where files bypass the server
  </h2>
</div>

<br />

<div align="center">

[![GitHub Release][release-img]][release]
[![Backend Quality][backend-img]][backend]
[![Backend Integration tests][integration-img]][integration]
[![Frontend Quality][frontend-img]][frontend]
[![Docker Build][docker-build-img]][docker-build]
[![License: Apache-2.0][license-img]][license]

</div>

![SafeBucket List View](./assets/list_view.png)

## Features

- Direct uploads and downloads via presigned URLs: files bypass the server
- Swappable infrastructure: every component (storage, database, events, cache, notifier) can be replaced
- SSO via any OIDC provider, with local auth for external users
- Role-based access control at platform and bucket level
- Quick/reverse share: share file via public links with options (password, max downloads, max views, etc...)
- Real-time activity tracking and audit logs
- Multifactor authentication (TOTP)
- File expiration, trash with configurable retention
- Admin dashboard with platform-wide statistics

And more... see the [full list of features](https://docs.safebucket.io/features).

## Architecture

![SafeBucket HLD](./assets/hld.png)

## Quick Start

```bash
git clone https://github.com/safebucket/safebucket.git
cd safebucket/deployments/local/lite
docker compose up -d
```

- Go to http://localhost:8080
- Log in with:
    - login: admin@safebucket.io
    - password: ChangeMePlease

> **Note:** If you are accessing Safebucket from an external machine (e.g. Proxmox), you need to update the following
> environment variables in the .env file with your host's IP or domain:
> - `STORAGE__RUSTFS__EXTERNAL_ENDPOINT`
> - `APP__ALLOWED_ORIGINS`
> - `APP__API_URL`
> - `APP__WEB_URL`

## Verify Image Signature

All published container images are signed with [cosign](https://github.com/sigstore/cosign) using keyless signing via
GitHub Actions OIDC: no manual keys are involved.

You can verify the signature of any published image using the following commands:

```bash
cosign verify \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  --certificate-identity-regexp=https://github.com/safebucket/safebucket/ \
  ghcr.io/safebucket/safebucket:<tag>
```

Replace `<tag>` with the image tag you want to verify (e.g., `latest`, `v1.0.0`).

## Star History

<a href="https://www.star-history.com/?type=date&legend=top-left&repos=safebucket%2Fsafebucket">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=safebucket/safebucket&type=date&theme=dark&legend=top-left&sealed_token=dHsVUeCTzOInC_KmU8r_6IlKegsMgz60XuyDH4mZ4hI7kvjW4mWJ1P3OhR50H_hzOLkuPqeT62gAOoVX8tiKV8qANvyEreqgJ1gCnJlkCP9rroF8NtqiiHO8GxxpRmTpTboXm1GFOLuxnOXIu8jwDVTffF_P0wXQ14JAl8_ipDL6kFsr3Jd3X7EBMiKN" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=safebucket/safebucket&type=date&legend=top-left&sealed_token=dHsVUeCTzOInC_KmU8r_6IlKegsMgz60XuyDH4mZ4hI7kvjW4mWJ1P3OhR50H_hzOLkuPqeT62gAOoVX8tiKV8qANvyEreqgJ1gCnJlkCP9rroF8NtqiiHO8GxxpRmTpTboXm1GFOLuxnOXIu8jwDVTffF_P0wXQ14JAl8_ipDL6kFsr3Jd3X7EBMiKN" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=safebucket/safebucket&type=date&legend=top-left&sealed_token=dHsVUeCTzOInC_KmU8r_6IlKegsMgz60XuyDH4mZ4hI7kvjW4mWJ1P3OhR50H_hzOLkuPqeT62gAOoVX8tiKV8qANvyEreqgJ1gCnJlkCP9rroF8NtqiiHO8GxxpRmTpTboXm1GFOLuxnOXIu8jwDVTffF_P0wXQ14JAl8_ipDL6kFsr3Jd3X7EBMiKN" />
 </picture>
</a>

## License

This project is licensed under the Apache 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with ❤️ using Go and React
- UI components by [Radix UI](https://radix-ui.com) and [shadcn/ui](https://ui.shadcn.com)
- Database ORM by [Gorm](https://gorm.io/index.html)
- Database migrations by [Goose](https://github.com/pressly/goose)
- Pub/sub integrations by [Watermill](https://watermill.io)
- Configuration management by [Koanf](https://github.com/knadh/koanf)
- Icons by [Lucide](https://lucide.dev)

[release]: https://github.com/safebucket/safebucket/releases

[release-img]: https://img.shields.io/github/v/release/safebucket/safebucket

[backend]: https://github.com/safebucket/safebucket/actions/workflows/quality-backend.yml

[backend-img]: https://github.com/safebucket/safebucket/actions/workflows/quality-backend.yml/badge.svg

[integration]: https://github.com/safebucket/safebucket/actions/workflows/integration-backend.yml

[integration-img]: https://github.com/safebucket/safebucket/actions/workflows/integration-backend.yml/badge.svg

[frontend]: https://github.com/safebucket/safebucket/actions/workflows/quality-frontend.yml

[frontend-img]: https://github.com/safebucket/safebucket/actions/workflows/quality-frontend.yml/badge.svg

[docker-build]: https://github.com/safebucket/safebucket/actions/workflows/docker-build.yml

[docker-build-img]: https://github.com/safebucket/safebucket/actions/workflows/docker-build.yml/badge.svg

[license]: https://github.com/safebucket/safebucket/blob/main/LICENSE

[license-img]: https://img.shields.io/github/license/safebucket/safebucket

[github-downloads-img]: https://img.shields.io/github/downloads/safebucket/safebucket/total

[docker-pulls]: https://img.shields.io/docker/pulls/safebucket/safebucket
