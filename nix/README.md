# Nixpkgs packaging

This directory holds a starter [`package.nix`](./package.nix) for submitting
nano-ffmpeg to [`NixOS/nixpkgs`](https://github.com/NixOS/nixpkgs).

It is **not** consumed by any in-repo build. The repository's own [`flake.nix`](../flake.nix)
is what powers `nix run` / local development. This file exists to track the
upstream packaging so the next nixpkgs bump is a copy-paste.

## Submitting to nixpkgs

### 1. Fork and check out nixpkgs

```bash
gh repo fork NixOS/nixpkgs --clone --remote
cd nixpkgs
git checkout -b nano-ffmpeg-init
```

### 2. Drop the file in place

```bash
mkdir -p pkgs/by-name/na/nano-ffmpeg
cp /path/to/nano-ffmpeg/nix/package.nix pkgs/by-name/na/nano-ffmpeg/package.nix
```

### 3. Compute real hashes

Both `src.hash` and `vendorHash` are placeholders (`lib.fakeHash`). Replace
them by triggering a build and copying the actual hash from the failure
message:

```bash
nix-build -A nano-ffmpeg
# Copy the "got: sha256-..." line into src.hash, re-run.
# Repeat for vendorHash.
```

Or compute the source hash up front:

```bash
nix-shell -p nix-prefetch-github --run \
  "nix-prefetch-github dgr8akki nano-ffmpeg --rev v0.5.0"
```

### 4. Add the maintainer entry

The package references `lib.maintainers.dgr8akki`, which must exist in
`maintainers/maintainer-list.nix`. Add (in alphabetical order):

```nix
dgr8akki = {
  name = "Aakash Pahuja";
  email = "pahujaaakash5@gmail.com";
  github = "dgr8akki";
  githubId = 17708157;
};
```

### 5. Validate locally

```bash
nix-build -A nano-ffmpeg
./result/bin/nano-ffmpeg --help
nix-shell -p nixpkgs-review --run "nixpkgs-review wip"
nix-shell -p nixfmt-rfc-style --run "nixfmt pkgs/by-name/na/nano-ffmpeg/package.nix"
```

### 6. Open the PR

Title convention: `nano-ffmpeg: init at 0.5.0`. Fill out the nixpkgs PR
template (platforms tested, `nixpkgs-review` output, etc.). Address review
nits and iterate.

## Updating on future releases

For each new tag of nano-ffmpeg, the upstream PR is just three edits:

1. Bump `version`.
2. Replace `src.hash` with `lib.fakeHash`, rebuild, paste the new hash.
3. Replace `vendorHash` only if `go.mod` changed; otherwise leave as-is.

Mirror the same edits into this directory so the next maintainer doesn't
have to re-derive them.
