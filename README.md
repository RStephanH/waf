# Web Application Firewall (WAF) Lab

This repo contains the docs and assets for a home lab focused on building and testing a Web Application Firewall (WAF) solution.

## Project Architecture

This repo is organized into 4 directories:

- `ubuntu`: bash script to provision a clean Ubuntu VM (base template)
- `waf`: script to set up the security solution (nginx + Coraza) on top of the Ubuntu VM
- `webserver`: script to set up the protected asset, an OWASP Juice Shop server
- `docs`: narrative documentation, including `journal.md`

## Network Topology

The lab uses two network interfaces per relevant VM:

- **NAT**: for internet access (package installs, updates)
- **Isolated network**: where the actual attack/defense traffic flows (Kali → WAF → Juice Shop)

This separation keeps the "battlefield" traffic contained and makes packet capture/analysis cleaner later on.

## Technology

### Virtualization & Provisioning

- QEMU with libvirt as the hypervisor environment
- cloud-init (NoCloud) for VM provisioning
- Ubuntu as the base OS for all VMs

### Role-Specific Technologies

- **Protected asset**: OWASP Juice Shop. The lab focuses on Layer 7 (HTTP) attacks, so a web app is the natural target.
- **WAF engine**: [Coraza](https://coraza.io/), an OWASP WAF engine written in Go, used as a drop-in ModSecurity-compatible alternative
- **Reverse proxy**: nginx, fronted by Coraza via its [SPOA (nginx) connector](https://github.com/corazawaf/coraza-spoa) or the `coraza-caddy`-style embedded approach (final connector choice to be confirmed in `waf/` implementation)
- **Ruleset**: OWASP Core Rule Set (CRS) — Coraza is CRS-compatible out of the box, so we get standard, battle-tested rules without having to write SecLang from scratch
- **Attacker host**: Kali Linux

### Why Coraza over ModSecurity

This lab intentionally chose Coraza (Go) over the more common ModSecurity (C++) for two reasons: it's an actively developed OWASP project, and it reinforces Go as the primary language across this portfolio (see FRED, connectit). ModSecurity remains the more "industry-default" choice and is noted here for context; it may be evaluated later as a comparison exercise.

### Snapshot Strategy

Each VM is snapshotted (`virsh snapshot-create-as`) once cleanly provisioned, to allow fast resets between attack simulations without re-running provisioning from scratch.

## Future Perspective (out of current scope)

The following are not implemented in this iteration due to hardware constraints, but are natural extensions of this lab:

- Centralized log pipeline (e.g. ELK/Suricata-style setup, as used in a separate IDS homelab project)
- Correlating WAF audit logs with network-level detection
- Side-by-side comparison against ModSecurity + CRS for the same attack set

## Documentation

Documentation lives in the `docs` directory. `journal.md` follows a narrative style, tracking the project's progress and decisions as they're made. This README stays a general, structural reference for the repository.
