# WAF Journey

We start this journey by reusing the clean Ubuntu VM template script from a previous homelab. Once that's confirmed working, it's time to plan.

## Planning

The project is split into small, sequential tasks.

### Tasks

- [x] Prepare and install prerequisites (hypervisor, cloud-localds package, etc.)
- [x] Get the Ubuntu VM script template
- [x] Choose the technology stack for each VM
- [ ] Design the network topology: two interfaces per VM, one NAT and one isolated
- [ ] Provision all VMs and confirm communication (isolated network + internet via NAT)
- [ ] Deploy nginx + Coraza + CRS in detection/log-only mode first
- [ ] Simulate attacks against Juice Shop through the WAF and observe Coraza's logs
- [ ] Compare results in detection vs blocking mode, tune the CRS paranoia level based on false positives/negatives

### Notes

Log analysis will stay minimal for now (raw Coraza logs, `tail -f` / `jq`). A full SIEM-style pipeline is a good idea for later, but not a goal of this lab given current hardware constraints.

## Task Details

### Environment Setup

Checked the ArchWiki (my daily driver is Arch) and cross-referenced with generative AI tools for the parts specific to QEMU/libvirt and cloud-init, since most official docs assume Debian/Ubuntu as the host.

### VM Base Image

As mentioned, there's already a bash script producing a clean Ubuntu VM in one command. The next step is extending it with the extra packages needed for this lab (see the script source for details).

> [!NOTE]
> Provisioning uses cloud-init, as mentioned above.

### Technology Stack Decision

Initially planned to use ModSecurity v3 as the WAF engine, since it's the historical standard and pairs naturally with nginx and the OWASP CRS. After comparing it against alternatives (Coraza, NAXSI, commercial options), we decided to go with **Coraza** instead.

The reasoning: Coraza is an official OWASP project, actively maintained, CRS-compatible, and written in Go — which is our main language going forward (Go first, Python second, C++ third for future IoT/low-level work). ModSecurity would have been the safer "industry-default" pick, but Coraza gives more long-term value: it's readable, hackable, and aligned with the stack we're actually building depth in across other projects (FRED, connectit).

Trade-off acknowledged: less mature documentation and community than ModSecurity, so expect more manual debugging during the `waf/` implementation phase.

Next step: confirm the exact Coraza deployment model (nginx connector vs SPOA vs embedded) before writing the setup script.
