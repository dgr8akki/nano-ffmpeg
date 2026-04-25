{
  lib,
  buildGoModule,
  fetchFromGitHub,
  makeWrapper,
  ffmpeg,
}:

buildGoModule (finalAttrs: {
  pname = "nano-ffmpeg";
  version = "0.5.0";

  src = fetchFromGitHub {
    owner = "dgr8akki";
    repo = "nano-ffmpeg";
    tag = "v${finalAttrs.version}";
    # To compute: replace with lib.fakeHash, run nix-build, copy the
    # "got:" hash from the error message back here.
    # Or: nix-prefetch-github dgr8akki nano-ffmpeg --rev v0.5.0
    hash = lib.fakeHash;
  };

  # To compute: replace with lib.fakeHash, run nix-build, copy the "got:"
  # hash from the vendor step's error message back here.
  vendorHash = lib.fakeHash;

  nativeBuildInputs = [ makeWrapper ];

  ldflags = [
    "-s"
    "-w"
    "-X github.com/dgr8akki/nano-ffmpeg/cmd.Version=${finalAttrs.version}"
  ];

  # ffmpeg/ffprobe are detected via exec.LookPath, so adding them to the
  # wrapped PATH is enough — no patching of source paths required.
  postInstall = ''
    wrapProgram $out/bin/nano-ffmpeg \
      --prefix PATH : ${lib.makeBinPath [ ffmpeg ]}
  '';

  meta = {
    description = "Beautiful keyboard-driven TUI for ffmpeg";
    longDescription = ''
      nano-ffmpeg wraps the full power of ffmpeg in a terminal dashboard.
      Browse files, pick an operation (convert, compress, trim, resize,
      extract audio, merge, subtitles), tweak settings with presets, and
      watch live progress while it encodes.
    '';
    homepage = "https://github.com/dgr8akki/nano-ffmpeg";
    changelog = "https://github.com/dgr8akki/nano-ffmpeg/releases/tag/v${finalAttrs.version}";
    license = lib.licenses.mit;
    mainProgram = "nano-ffmpeg";
    maintainers = with lib.maintainers; [ dgr8akki ];
    platforms = lib.platforms.unix;
  };
})
