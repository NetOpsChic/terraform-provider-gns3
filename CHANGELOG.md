# v3.0.0 Release Notes

## Highlights

- New `gns3_nat` resource
- `symbol` (icon) support added across all node resources
- Significant reliability fixes to `Read`/`Update`/`Delete` across six resource files, found and confirmed through live testing against a real GNS3 server rather than code review alone
- One confirmed upstream GNS3 API bug filed: [GNS3/gns3-server#2811](https://github.com/GNS3/gns3-server/issues/2811)
- Dead code removed (`utiles.go`)
- **Three breaking changes** — see the dedicated section below before upgrading

This release involved substantially more testing than usual — every fix below was validated against a live GNS3 instance (state inspection, direct API `curl` checks, and GUI confirmation), not just compiled and assumed correct. Several bugs were only caught this way; multiple resources previously reported success on operations that silently did nothing.

---

## ⚠ Breaking Changes

Three changes in this release are breaking under semver. All three are fixes to functionality that was already broken or non-functional — none of them change behavior that previously worked correctly.

**1. `data.gns3_link_id` — arguments completely changed**
- Before: `name` (string)
- Now: `node_a_id`, `node_a_adapter`, `node_a_port`, `node_b_id`, `node_b_adapter`, `node_b_port`
- Why: the old version called a URL that does not exist in GNS3's API and matched against a `name` field that links don't have. It could never have returned a successful result for any input, ever. Any existing config using the old `name` argument was already non-functional.

**2. `gns3_nat.name` is now `ForceNew`**
- Before: changing `name` called `Update`, reported success, and silently did nothing — GNS3 accepts the rename request but never applies it (confirmed upstream bug, [GNS3/gns3-server#2811](https://github.com/GNS3/gns3-server/issues/2811)).
- Now: changing `name` destroys and recreates the NAT node.
- Impact: if a NAT node has links attached, recreating it may also require recreating those links. Review your plan output carefully before applying if you rename an existing `gns3_nat` resource.

**3. `gns3_link`'s six endpoint fields are now `ForceNew`**
- Fields: `node_a_id`, `node_a_adapter`, `node_a_port`, `node_b_id`, `node_b_adapter`, `node_b_port`
- Before: changing any of these called `Update`, reported success, and (confirmed via live testing) did not actually rewire the link on GNS3's side.
- Now: any change to these fields destroys and recreates the link.

**Upgrade guidance:** run `terraform plan` after upgrading and review it carefully before applying. If your existing configuration only creates and destroys these resources (rather than editing them in place), you will see no behavioral difference. The risk is specifically for configs that previously edited a NAT node's name or a link's endpoints in place — those "successful" applies were not actually doing what they claimed.

---

## New: `gns3_nat` resource

Manage GNS3 NAT nodes as first-class Terraform resources.

```hcl
resource "gns3_nat" "nat1" {
  project_id = gns3_project.lab.project_id
  name       = "NAT1"
  x          = 800
  y          = 500
  symbol     = ":/symbols/computer.svg"
}
```

**Known limitation:** GNS3's API silently ignores rename requests for NAT nodes specifically — it returns `200 OK` and echoes the requested name, but the change is never persisted. Confirmed via direct API testing and filed upstream as [GNS3/gns3-server#2811](https://github.com/GNS3/gns3-server/issues/2811). Because of this, **`name` is `ForceNew`** — changing it destroys and recreates the NAT node rather than silently failing to rename it in place. This behavior is specific to `nat`; the identical rename operation works correctly for `switch`, `cloud`, `project`, and `dynamips`-type template nodes (all confirmed via live testing).

---

## New: `symbol` (icon) support

All node resources now support a `symbol` argument to set the node's canvas icon, matching GNS3's own `symbol_id` values (e.g. `:/symbols/computer.svg`, `:/symbols/classic/atm_switch.svg`). Use `GET /v2/symbols` on your GNS3 server to list all available icons.

Added to: `gns3_nat`, `gns3_switch`, `gns3_cloud`, `gns3_docker`, `gns3_qemu_node`.

```hcl
resource "gns3_switch" "switch1" {
  ...
  symbol = ":/symbols/classic/atm_switch.svg"
}
```

`symbol` is `Optional` + `Computed` — if you don't set it, GNS3's default icon for that node type is used and won't show as drift. Confirmed working via live create, drift-detection, and update-revert testing on `gns3_switch`, then rolled out identically to the other four resources.

**Not included in this release:** uploading a *custom* icon file (`POST /v2/symbols/{symbol_id}/raw`). This release only lets you *reference* an existing symbol (built-in or already uploaded); custom icon upload would be a separate resource with a different request shape (raw file bytes, not JSON) and is a candidate for a future release.

---

## Fixed: `resource_switch.go`, `resource_cloud.go`

Both files were copy-pasted from an incomplete template and shared two bugs:

- **`Read` was a stub.** It only checked whether the node still existed (200 vs 404) and never decoded the response body — meaning `name`, `compute_id`, `x`, `y` were never synced back into Terraform state. Renaming a switch or cloud node in the GNS3 GUI, or moving it, would never show up as drift. **Fixed:** full field decode and sync.
- **`Delete` didn't tolerate a 404.** If the node had already been deleted manually, `terraform destroy` would fail instead of treating "already gone" as success. **Fixed.**

**Confirmed via live testing:** unlike `gns3_nat`, rename works correctly for both `switch` and `cloud` node types — confirmed by direct `curl` PUT before writing the fix, so no `ForceNew` was needed here.

---

## Fixed: `resource_docker.go`

This file had the most bugs of any resource in the provider.

- **`Delete` didn't check the response status at all** — only a transport-level error (`err != nil`) was checked; any HTTP failure status (403, 500, etc.) was silently treated as a successful delete. This was more serious than a missing 404-tolerance check elsewhere — it meant `terraform destroy` could report success while the container still existed. **Fixed:** status code now checked, with `204`/`404` treated as success.
- **`Read` was a stub** — same class of bug as switch/cloud. **Fixed:** full sync of `name`, `compute_id`, `x`, `y`, `image`, `extra_volumes`, `start_command`, and `start` (derived from node status).
- **`environment` and `console_type` were nested under `properties` in the update payload, but GNS3 treats them as top-level `Node` attributes.** Confirmed against the documented GNS3 node schema and against a real user-reported fix for the identical issue in `resource_qemu_vm.go` (which already had this right). The mis-nesting meant updates to these fields were silently accepted by the API but never applied. **Fixed:** both moved to top-level fields on the `DockerNode` struct, matching how `qemu` already handled them correctly.
  - **Confirmed via live testing:** attempting to set `console_type = "vnc"` correctly returned a `409` from GNS3 ("Please install TigerVNC server") — proving the request now reaches the correct field (previously, a mis-nested `console_type` would have been silently ignored with a `200 OK`, not rejected). Setting `console_type = "none"` succeeded and was confirmed via direct API check.
- **`start_command` update was dead code** — the field was appended to `updateData` *after* the HTTP request had already been marshaled and sent, so changes to it had zero effect. **Fixed:** moved into the request body construction before the request is sent.
- **`start` only took effect at node creation** — there was no way to stop/start an existing container by changing `start` in config. **Fixed:** implemented as a real toggle via the `/start` and `/stop` node lifecycle endpoints.
- `io/ioutil` → `io`; unchecked `d.Set` errors fixed throughout.

---

## Fixed: `resource_qemu_vm.go`

- **`start_vm = false` was silently ignored if the VM was already running** — this was the most serious bug found this session. The final running-state decision used `wasRunning || d.Get("start_vm").(bool)`, meaning a VM that was running before the update would always be restarted regardless of the user's declared `start_vm = false`. `terraform apply` reported success; the VM kept running. **Fixed:** the final state is now driven solely by `d.Get("start_vm")`, confirmed live — setting `start_vm: true → false` on a running node now correctly stops it (verified via direct API status check).
- **`Read` only synced 3 fields** (`name`, `x`, `y`) out of roughly 15 available. Changes to `ram`, `cpus`, `adapter_type`, `mac_address`, `hda_disk_image`, and others were invisible to drift detection. **Fixed:** full decode of the `properties` object and top-level `console`/`console_type`, using new helper functions (`setIntField`, `setStringField`) to handle Go's JSON-number-to-`float64` decoding correctly. **Confirmed via live testing:** changed `ram` directly via the API, bypassing Terraform, and confirmed `terraform plan` now correctly shows the resulting drift (previously silent).
- **`mac_address` caused a permanent phantom diff** — a bug introduced by fixing `Read` above. GNS3 auto-generates a MAC address when none is specified; once `Read` started syncing it back, the schema's missing `Computed: true` meant Terraform tried to null it out on every single `plan`/`apply`, forever. **Fixed:** added `Computed: true`, matching the treatment `console` already correctly had. Confirmed via a clean `terraform plan` → "No changes" afterward.
- `io/ioutil` → `io`; `Delete` now tolerates 404.

**Already correct, no change needed:** `console`/`console_type` were already handled as top-level fields (not nested under `properties`), consistent with a prior fix from another contributor.

---

## Fixed: `resource_link.go`

- **Wrong success status code in `Update`** — the code checked for `200 OK`, but GNS3's documented API returns `201` for a successful link update. **Fixed:** both accepted, defensively.
- **`Read` was a stub.** **Fixed:** full decode and sync of both endpoints (`node_a_id`/`adapter`/`port`, `node_b_id`/`adapter`/`port`), including logic to correctly match array ordering against existing state rather than assuming the API always returns endpoints in the same order they were created with.
- **GNS3 does not actually support rewiring an existing link's endpoints, despite accepting the request and returning success.** This is the most significant finding for this file, discovered through live testing: after applying a config change to `node_a_port`, the API returned success and `Read` correctly reflected the change in Terraform state — but a direct API check confirmed the port on GNS3's side had **not** actually changed. This is the identical failure pattern to the NAT rename bug, on a different resource. **Fixed:** all six endpoint-defining fields (`node_a_id`, `node_a_adapter`, `node_a_port`, `node_b_id`, `node_b_adapter`, `node_b_port`) are now `ForceNew` — Terraform destroys and recreates the link instead of attempting a silent no-op update. **Confirmed via live testing:** a destroy+recreate cycle correctly produced a new `link_id` and the new port assignment was verified as actually applied via direct API check.
- `waitForNode` (used during link creation to poll until both endpoint nodes are registered) now uses a client with a 15s timeout instead of the default unbounded client; `io/ioutil` → `io`.

**Candidate for a second upstream GNS3 issue** (not yet filed): the same "PUT accepted, returns success, doesn't apply" pattern as NAT rename, but for link endpoint rewiring.

---

## Fixed: `data_source_link_id.go`

This data source was **non-functional since introduction** — two independent bugs meant it could never successfully return a result:

- **Endpoint URL was fabricated.** It called `/v2/controller/link/projects/{project_id}/links`, which does not exist in GNS3's API (confirmed against official documentation). The real endpoint is `/v2/projects/{project_id}/links`.
- **Matched on a `name` field that links do not have.** GNS3 link objects have no `name` — they're identified only by `link_id` and by which node/adapter/port pairs they connect. The original matching logic (`link["name"] == linkName`) could never succeed.

**This is a breaking schema change** — the data source's arguments changed from a single `name` string to six endpoint-identifying fields (`node_a_id`, `node_a_adapter`, `node_a_port`, `node_b_id`, `node_b_adapter`, `node_b_port`), since `name` had no real value to preserve. Given the data source never worked under any input, this is unlikely to affect real configurations, but is called out here for completeness.

```hcl
data "gns3_link_id" "my_link" {
  project_id     = gns3_project.lab.project_id
  node_a_id      = gns3_switch.sw1.id
  node_a_adapter = 0
  node_a_port    = 0
  node_b_id      = gns3_nat.nat1.id
  node_b_adapter = 0
  node_b_port    = 0
}
```

**Confirmed via live testing:** both orderings of the two endpoints tested (i.e. specifying the same link with `node_a`/`node_b` swapped) — both correctly resolved to the identical `link_id`.

---

## Fixed: `resource_project.go`

- **`Delete` did not check the HTTP response status at all** — only the Go-level transport error was checked; the response was discarded entirely. Any failure status from GNS3 (e.g. a conflict, a server error) would be silently treated as a successful project deletion, and the response body was never closed. **Fixed:** status code now checked (`204`/`404` = success), response body properly closed, error message includes the response body on failure.
- HTTP client timeout added throughout (`Create`, `Read`, `Update`, `Delete` now share a single client with a 15s timeout, replacing a mix of bare `http.Get`/`http.Post`/`http.DefaultClient`); `io/ioutil` → `io`; unchecked `d.Set` errors fixed.

**Already correct, no change needed:** project rename via `Update` — confirmed working live earlier in this cycle (`test-nat` → `test-all-resources` persisted correctly), unlike NAT node rename.

---

## Fixed: `resource_templates.go`

- **`Read` was a stub.** **Fixed:** full sync of `name`, `compute_id`, `x`, `y`, and `start` (derived from node status). **Confirmed via live testing:** directly modified a template-instantiated node's position via the API and confirmed `terraform plan` correctly detected the resulting drift (previously silent), and that the position correction from `Update` genuinely wrote back to GNS3, not just Terraform state.
- **`start` had no update path** — it only took effect at creation time. **Fixed:** implemented as a real start/stop toggle in `Update`, via the node lifecycle endpoints.
- HTTP client timeout added; `io/ioutil` → `io`; unchecked `d.Set` errors fixed.

**Note on scope:** this resource instantiates a node from a GNS3 template, and the resulting node's type (`qemu`, `dynamips`, `docker`, `nat`, etc.) depends entirely on which template is used — unlike other resources in this provider where the node type is fixed. Because we've confirmed GNS3 silently ignores renames specifically for `nat`-type nodes, a template that happens to instantiate a NAT appliance could theoretically hit the same issue via this resource. This has not been separately tested for every possible template type; the full `Read` sync means any such silent-rename-ignored case will at least now show up as persisting drift in `terraform plan`, rather than being silently hidden as it was before this fix.

---

## Documentation only (no functional change): `resource_start_all.go`

Reviewed and confirmed the endpoint and status-code handling were already correct (`/v2/projects/{project_id}/nodes/start`, `204` on success). Added:
- HTTP client timeout, response body included in error messages
- Explicit documentation of two behaviors that were previously implicit and undocumented:
  1. This resource has **no drift detection** — if nodes are later stopped, `terraform plan` will not notice or re-trigger the start action.
  2. `terraform destroy` on this resource does **not** stop any nodes — it only removes the resource from Terraform state.

---

## Removed: `provider/utiles.go`

Contained two unexported helper functions (`getProjectID`, `getTemplateID`) that were confirmed dead code — not called anywhere in the package (confirmed via `go build`'s "declared and not used" style errors after removal, and via `grep` across the codebase beforehand). Deleted entirely rather than left as unused code.

---

## Upstream GNS3 issues filed

- **[GNS3/gns3-server#2811](https://github.com/GNS3/gns3-server/issues/2811)** — `PUT /v2/projects/{project_id}/nodes/{node_id}` silently ignores name changes for `nat`-type nodes. Filed with a minimal `curl`-based reproduction comparing against working `switch`/`cloud` rename behavior.

---

## Known limitations carried into this release

- `gns3_nat.name` is `ForceNew` due to the confirmed upstream GNS3 bug above.
- `gns3_link`'s six endpoint fields are all `ForceNew` — GNS3 does not support rewiring an existing link's connections in place, despite the API accepting the request.
- `gns3_docker`'s `environment` field is not synced back in `Read` — GNS3 returns it as a flattened string rather than the map Terraform expects, and round-tripping it correctly would need additional parsing not included in this release.
- `gns3_start_all` has no drift detection and does not stop nodes on destroy (see above) — this is a structural limitation of modeling an action as a resource under the current Terraform Plugin SDK v2, not something fixable without a larger redesign (e.g. a `triggers`-style map, or migrating to the Plugin Framework's action resources).
- Migration to the Terraform Plugin Framework (tracked separately, not part of this release) would resolve some of the above more structurally, but is a substantial, separate effort — recommended to be done incrementally via `terraform-plugin-mux` rather than as a single release.

---

## Version

**2.5.5 → 3.0.0** (major). Three changes in this release are breaking under strict semver — a changed data source schema and two new `ForceNew` behaviors (see the Breaking Changes section above) — even though all three are corrections to functionality that was already non-functional or silently incorrect rather than changes to working behavior.