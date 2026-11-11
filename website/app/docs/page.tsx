export default function DocsPage() {
  return (
    <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-16">
      <div className="flex gap-12">
        {/* Sidebar */}
        <aside className="hidden lg:block w-56 shrink-0">
          <nav className="sticky top-24 space-y-1">
            <SidebarLink href="#getting-started">Getting Started</SidebarLink>
            <SidebarLink href="#screen-flow">Screen Flow</SidebarLink>
            <SidebarLink href="#operations">Operations</SidebarLink>
            <SidebarLink href="#keybindings">Keybindings</SidebarLink>
            <SidebarLink href="#configuration">Configuration</SidebarLink>
            <SidebarLink href="#progress">Progress Screen</SidebarLink>
            <SidebarLink href="#smart-defaults">Smart Defaults</SidebarLink>
          </nav>
        </aside>

        {/* Content */}
        <article className="flex-1 min-w-0">
          <h1 className="text-4xl font-bold text-white mb-2">Documentation</h1>
          <p className="text-[#9CA3AF] mb-12">
            Everything you need to know about using nano-ffmpeg.
          </p>

          {/* Getting Started */}
          <Section id="getting-started" title="Getting Started">
            <p>
              nano-ffmpeg wraps ffmpeg in a keyboard-driven terminal dashboard.
              Install it and run:
            </p>
            <CodeBlock lines={["$ nano-ffmpeg"]} />
            <p>
              You can also force the UI theme for a single run:
            </p>
            <CodeBlock
              lines={[
                "$ nano-ffmpeg --theme dark",
                "$ nano-ffmpeg --theme light",
              ]}
            />
            <p>
              You can also control the startup path:
            </p>
            <CodeBlock
              lines={[
                "# Start File Picker in this directory",
                "$ nano-ffmpeg --dir /path/to/folder",
                "",
                "# Skip File Picker and preload this file",
                "$ nano-ffmpeg --dir /path/to/video.mp4",
              ]}
            />
            <p>
              That&apos;s it. The TUI guides you through file selection, operation
              picking, settings configuration, and encoding. You need{" "}
              <Code>ffmpeg</Code> and <Code>ffprobe</Code> installed. For full
              Stabilize support, use an ffmpeg build with vidstab
              (Homebrew: <Code>ffmpeg-full</Code>).
            </p>
            <p>
              If you install nano-ffmpeg via the Homebrew tap,{" "}
              <Code>ffmpeg-full</Code> is installed as a dependency.
              On Windows, the Scoop manifest declares <Code>extras/ffmpeg</Code>{" "}
              as a dependency so ffmpeg is pulled automatically.
            </p>
            <h3 className="text-lg font-semibold text-white mt-6 mb-3">
              Install nano-ffmpeg
            </h3>
            <CodeBlock
              lines={[
                "# macOS / Linux (Homebrew)",
                "$ brew install dgr8akki/tap/nano-ffmpeg",
                "",
                "# Windows (Scoop)",
                "> scoop bucket add extras",
                "> scoop bucket add nano-ffmpeg https://github.com/dgr8akki/scoop-bucket",
                "> scoop install nano-ffmpeg",
                "",
                "# Any OS with Go",
                "$ go install github.com/dgr8akki/nano-ffmpeg@latest",
              ]}
            />
            <h3 className="text-lg font-semibold text-white mt-6 mb-3">
              Install ffmpeg separately (if not using Homebrew tap / Scoop)
            </h3>
            <CodeBlock
              lines={[
                "# macOS",
                "$ brew install ffmpeg-full",
                "",
                "# macOS (minimal build; Stabilize uses deshake fallback)",
                "$ brew install ffmpeg",
                "",
                "# Ubuntu / Debian",
                "$ sudo apt install ffmpeg",
                "",
                "# Fedora",
                "$ sudo dnf install ffmpeg",
                "",
                "# Windows (Scoop)",
                "> scoop bucket add extras",
                "> scoop install extras/ffmpeg",
                "",
                "# Windows (winget)",
                "> winget install ffmpeg",
              ]}
            />
          </Section>

          {/* Screen Flow */}
          <Section id="screen-flow" title="Screen Flow">
            <p>nano-ffmpeg uses a multi-screen navigation pattern:</p>
            <div className="my-6 p-4 bg-[#111827] rounded-lg border border-[#1f2937] font-mono text-sm text-center">
              <span className="text-[#7C3AED]">Home</span>
              <span className="text-[#6B7280]"> → </span>
              <span className="text-[#06B6D4]">File Picker</span>
              <span className="text-[#6B7280]"> → </span>
              <span className="text-[#7C3AED]">Operations</span>
              <span className="text-[#6B7280]"> → </span>
              <span className="text-[#06B6D4]">Settings</span>
              <span className="text-[#6B7280]"> → </span>
              <span className="text-[#7C3AED]">Progress</span>
              <span className="text-[#6B7280]"> → </span>
              <span className="text-[#22C55E]">Result</span>
            </div>
            <ul className="space-y-2">
              <Li><strong>Home</strong> — ffmpeg version, capabilities, recent files, operation list</Li>
              <Li><strong>File Picker</strong> — Browse filesystem or type a path. Inline ffprobe metadata preview. Use <Code>--dir &lt;directory&gt;</Code> to change the startup directory.</Li>
              <Li><strong>Direct file startup</strong> — Use <Code>--dir &lt;file&gt;</Code> to skip File Picker and start directly in Operations with the file preloaded.</Li>
              <Li><strong>Operations</strong> — Choose from 12 operations</Li>
              <Li><strong>Settings</strong> — Presets + individual knobs. Live command preview.</Li>
              <Li><strong>Progress</strong> — Live progress bar, ETA, stats, scrollable log</Li>
              <Li><strong>Result</strong> — Output path, size comparison, do another or quit</Li>
            </ul>
            <p className="mt-4">
              Press <Code>Esc</Code> on any screen to go back. Press{" "}
              <Code>q</Code> to quit.
            </p>
          </Section>

          {/* Operations */}
          <Section id="operations" title="Operations">
            <div className="space-y-4">
              <OpDoc name="Convert Format" desc="Change container format (MP4, MKV, WebM, AVI, MOV) and video codec (H.264, H.265, AV1, VP9). Quality presets from High to Tiny, preset speed slow/medium/fast/ultrafast, and audio codec (copy, AAC, MP3, Opus). Picks the container can't hold are swapped with a notice: WebM uses VP9 + Opus, MOV uses H.264 instead of AV1/VP9." />
              <OpDoc name="Extract Audio" desc="Strip video track, keep audio. Output to MP3, AAC, FLAC, WAV, OGG, or Opus. Bitrate presets from CD Quality (320k) down to Lo-fi (64k). The chosen format maps to a fixed encoder; there is no automatic stream copy today. FLAC and WAV ignore the bitrate. OGG uses Opus when your ffmpeg has no libvorbis (Homebrew's default build)." />
              <OpDoc name="Resize / Scale" desc="Scale to 4K, 1080p, 720p, 480p, or 360p height (width is auto-derived). Width follows the source aspect ratio, rounded to an even number. Codec choice: H.264 or H.265." />
              <OpDoc name="Trim / Cut" desc="Set start and end time as HH:MM:SS, MM:SS or seconds. Lossless cut (stream copy) toggle; lossless cuts start on the nearest keyframe. End time is pre-filled with the input's total duration from ffprobe. Invalid times, an end before the start, or a start past the end are flagged before anything runs." />
              <OpDoc name="Compress" desc="CRF quality: Visually Lossless (18) to Heavy (32). Codec choice: H.264, H.265, AV1. Preset speed: slow/medium/fast." />
              <OpDoc name="Merge / Concat" desc="Joins every file in the input's folder that shares the same extension, in alphabetical order, using ffmpeg's concat demuxer. Settings lists the files before you run it, and the merge output is never fed back in. Names with spaces, quotes or accents work. Stream-copy or re-encode to H.264/AAC. No reorder UI yet." />
              <OpDoc name="Add Subtitles" desc="Burn-in (hardcode) or soft-embed subtitle streams already present in the input file. Pick which subtitle track to use. Burn-in needs an ffmpeg build with libass (ffmpeg-full on macOS); without it, or when the file has no subtitle tracks, Settings says so and won't run. Font, size, and position customization are not yet exposed." />
              <OpDoc name="Create GIF" desc="Frame rate (10/15/24 fps), width presets, palette generation and paletteuse for quality, start time and duration selection. Output is GIF only today; WebP output is planned." />
              <OpDoc name="Extract Thumbnails" desc="Single frame at a timestamp (defaults to mid-clip for clips up to 10s), 4x4 contact sheet, or one frame every 5 seconds written as name_extract_thumbnails_001.png, _002.png, and so on. Output is PNG." />
              <OpDoc name="Watermark" desc="Overlays a solid white color box. Five positions (four corners and center), opacity (25/50/75%), and size presets (small/medium/large). Image and text watermarks are planned." />
              <OpDoc name="Audio Adjustments" desc="Normalize (loudnorm), volume up/down by 3, 6 or 10 dB, fade in + fade out (2s, auto-clamped to input duration), or remove audio entirely." />
              <OpDoc name="Video Filters" desc="Stabilize (vidstab when available; falls back to deshake with a warning in Settings), deinterlace (yadif), speed 2x/0.5x, rotate 90°, horizontal/vertical flip. Crop and color filters are planned." />
            </div>
          </Section>

          {/* Keybindings */}
          <Section id="keybindings" title="Keybindings">
            <h3 className="text-lg font-semibold text-white mb-3">Global</h3>
            <KeyTable keys={[
              ["q", "Quit (stops a running encode first)"],
              ["Ctrl+C", "Force quit"],
              ["?", "Toggle help overlay"],
            ]} />
            <p className="mt-3">
              While a text field or the path input has focus, <Code>q</Code> and{" "}
              <Code>?</Code> are typed as characters.
            </p>

            <h3 className="text-lg font-semibold text-white mt-6 mb-3">Navigation</h3>
            <KeyTable keys={[
              ["\u2191 / k", "Move up"],
              ["\u2193 / j", "Move down"],
              ["Enter", "Select / confirm / execute"],
              ["Esc", "Go back one screen"],
            ]} />

            <h3 className="text-lg font-semibold text-white mt-6 mb-3">File Picker</h3>
            <KeyTable keys={[
              ["Enter", "Open directory / select file"],
              ["Backspace", "Parent directory"],
              ["/", "Toggle path input mode"],
            ]} />

            <h3 className="text-lg font-semibold text-white mt-6 mb-3">Settings</h3>
            <KeyTable keys={[
              ["\u2190 / \u2192", "Change select/toggle value or move the text cursor"],
              ["Typing", "Edit text fields (Start/End Time, Duration, Timestamp)"],
              ["Enter", "Execute ffmpeg command"],
            ]} />

            <h3 className="text-lg font-semibold text-white mt-6 mb-3">Progress</h3>
            <KeyTable keys={[
              ["Esc", "Cancel (with confirmation); on a failed job, back to Settings"],
              ["y / n", "Confirm or deny cancellation"],
            ]} />
          </Section>

          {/* Configuration */}
          <Section id="configuration" title="Configuration">
            <p>
              Config is stored at <Code>~/.config/nano-ffmpeg/config.json</Code>:
            </p>
            <CodeBlock
              lines={[
                '{',
                '  "default_output_dir": "",',
                '  "theme": "dark",',
                '  "recent_files": [],',
                '  "hw_accel": "auto",',
                '  "ffmpeg_path": ""',
                '}',
              ]}
            />
            <div className="mt-4 space-y-2">
              <Li><Code>default_output_dir</Code> — Where output files are saved (empty = same as input; must be an existing directory)</Li>
              <Li><Code>theme</Code> — Color theme (<Code>dark</Code> or <Code>light</Code>)</Li>
              <Li><Code>recent_files</Code> — Last 10 files used (auto-populated)</Li>
              <Li><Code>hw_accel</Code> — Reserved; not applied to encodes yet</Li>
              <Li><Code>ffmpeg_path</Code> — Reserved; not read yet (ffmpeg is found on PATH, next to the binary, or in Homebrew&apos;s ffmpeg/ffmpeg-full kegs)</Li>
            </div>
            <p className="mt-4">
              Passing <Code>--theme dark|light</Code> overrides the config theme for that run.
            </p>
            <p className="mt-4">
              Passing <Code>--dir &lt;directory|file&gt;</Code> overrides startup location for that run.
            </p>
            <p className="mt-4">
              Capabilities are cached separately at{" "}
              <Code>~/.config/nano-ffmpeg/capabilities.json</Code> and
              auto-invalidated when your ffmpeg version changes.
            </p>
          </Section>

          {/* Progress Screen */}
          <Section id="progress" title="Progress Screen">
            <p>The progress screen parses ffmpeg&apos;s stderr in real-time:</p>
            <ul className="space-y-2 mt-4">
              <Li><strong>Gradient progress bar</strong> — green (0%) to cyan (100%), adapts to terminal width, measured against the expected output length (trims, GIFs and speed changes reach 100%)</Li>
              <Li><strong>ETA</strong> — Smoothed with rolling average over last 5 updates to avoid jitter</Li>
              <Li><strong>Stats</strong> — Elapsed, speed, FPS, frames, output size, bitrate</Li>
              <Li><strong>Braille spinner</strong> — For indeterminate operations (stream copy, concat)</Li>
              <Li><strong>Live log</strong> — Scrollable raw ffmpeg output, last 6 lines visible</Li>
              <Li><strong>Cancel</strong> — Esc opens confirmation; y stops ffmpeg (SIGINT, force-killed after 3s; killed directly on Windows) and removes the partial output. Quitting mid-encode does the same</Li>
              <Li><strong>Errors</strong> — A failed job shows the first ffmpeg error line plus a plain-English explanation; Esc returns to Settings</Li>
            </ul>
          </Section>

          {/* Smart Defaults */}
          <Section id="smart-defaults" title="Smart Defaults">
            <p>
              nano-ffmpeg runs <Code>ffprobe</Code> on your input file and uses
              parts of the result to avoid dead ends. The behaviors below are
              what&apos;s actually wired up today:
            </p>
            <div className="mt-4 space-y-2">
              <Li>Trim → the End Time field is pre-filled with the input&apos;s total duration</Li>
              <Li>Add Subtitles → the Subtitle Track selector lists the subtitle streams detected in the input (codec and language, when present); if none are found, Settings says so and won&apos;t run</Li>
              <Li>Audio Adjustments (fade) → the fade-out start is clamped to the input duration so short clips don&apos;t produce an invalid filter</Li>
              <Li>Video Filters → if vidstab is unavailable, Stabilize automatically uses <Code>deshake</Code> and shows a fallback warning in Settings</Li>
              <Li>Convert → codecs a container can&apos;t hold are swapped (WebM → VP9 + Opus, MOV → H.264) with a notice</Li>
              <Li>Extract Audio / Audio Adjustments → blocked with a message when the file has no audio track</Li>
              <Li>Extract Thumbnails → the timestamp defaults to mid-clip for clips up to 10s</Li>
              <Li>File Picker → shows an inline ffprobe preview (format, duration, size, video codec/resolution/fps, audio codec/channels/sample-rate, subtitle track count) while you&apos;re browsing</Li>
            </div>
            <p className="mt-4">
              Not done yet: suggesting an output codec from the input, graying
              out upscale targets, and pre-selecting AAC for stream copy.
            </p>
          </Section>
        </article>
      </div>
    </div>
  );
}

function SidebarLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      className="block text-sm text-[#6B7280] hover:text-white py-1.5 px-3 rounded-md hover:bg-[#111827] transition-colors"
    >
      {children}
    </a>
  );
}

function Section({
  id,
  title,
  children,
}: {
  id: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section id={id} className="mb-16 scroll-mt-24">
      <h2 className="text-2xl font-bold text-white mb-6 pb-3 border-b border-[#1f2937]">
        {title}
      </h2>
      <div className="text-[#d1d5db] leading-relaxed space-y-4">{children}</div>
    </section>
  );
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="px-1.5 py-0.5 rounded bg-[#1f2937] text-[#06B6D4] text-sm font-mono">
      {children}
    </code>
  );
}

function CodeBlock({ lines }: { lines: string[] }) {
  return (
    <div className="bg-[#111827] border border-[#1f2937] rounded-lg p-4 font-mono text-sm overflow-x-auto">
      {lines.map((line, i) => (
        <div key={i} className={line === "" ? "h-3" : "text-[#d1d5db]"}>
          {line}
        </div>
      ))}
    </div>
  );
}

function Li({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-2">
      <span className="mt-2 h-1.5 w-1.5 rounded-full bg-[#7C3AED] shrink-0" />
      <span>{children}</span>
    </li>
  );
}

function OpDoc({ name, desc }: { name: string; desc: string }) {
  return (
    <div className="p-4 rounded-lg border border-[#1f2937] bg-[#0d0d14]">
      <h4 className="text-sm font-semibold text-white mb-1">{name}</h4>
      <p className="text-sm text-[#9CA3AF]">{desc}</p>
    </div>
  );
}

function KeyTable({ keys }: { keys: string[][] }) {
  return (
    <div className="border border-[#1f2937] rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-[#111827]">
            <th className="text-left px-4 py-2 text-[#6B7280] font-medium w-32">Key</th>
            <th className="text-left px-4 py-2 text-[#6B7280] font-medium">Action</th>
          </tr>
        </thead>
        <tbody>
          {keys.map(([key, action], i) => (
            <tr key={i} className="border-t border-[#1f2937]">
              <td className="px-4 py-2">
                <code className="px-1.5 py-0.5 rounded bg-[#1f2937] text-[#06B6D4] text-xs font-mono">
                  {key}
                </code>
              </td>
              <td className="px-4 py-2 text-[#d1d5db]">{action}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
