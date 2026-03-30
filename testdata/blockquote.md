# Blockquote Test

Navidown is a reusable navigable markdown component library for Go that renders
markdown to ANSI for terminal output with scrolling, pager-style navigation,
interactive link traversal, and inline image display via the Kitty graphics protocol.

## Simple blockquote

The renderer converts markdown into decorated terminal lines. Each line carries
ANSI escape sequences for color, bold, italic, and other styling attributes.

> This is a simple blockquote that should render in a muted gray color,
> distinct from the main body text.

The output can be displayed in any terminal that supports ANSI escape codes.
Both 256-color and true-color profiles are supported depending on the terminal.

## Blockquote with emphasis

Glamour injects invisible zero-width markers during rendering. These markers
are later extracted to reliably map elements like headers and links to their
line positions in the rendered output.

> This blockquote has **bold text**, *italic text*, and `inline code` inside it.

The marker correlator achieves 100% reliability for position mapping. A scoring
correlator serves as a heuristic fallback when no markers are found.

## Nested blockquotes

Images flow through a deferred token system. During glamour rendering, image
elements emit placeholder tokens instead of rendering inline. A post-processor
then replaces these tokens with the actual display content.

> Outer blockquote.
>
> > Inner nested blockquote.
> > Should also be muted.
>
> Back to outer.

SVG images are rasterized to PNG via the resvg CLI tool. Mermaid diagrams are
rendered through the mmdc CLI. Both are cached by content hash to avoid
redundant processing on subsequent renders.

## Blockquote with a link

Two tview adapters wrap the core MarkdownSession. BoxViewer extends tview.Box
for low-level tcell drawing with custom ANSI color handling. TextViewViewer
extends tview.TextView and leverages its native paging and grapheme clustering.

> Check out the [project homepage](https://example.com) for more details.

Kitty image protocol support uses Unicode placeholder lines with combining
diacritics. The diacritics table must have exactly 297 entries from Unicode 6.0.0
combining class 230 characters.

## Blockquote with a list

Content providers implement a simple interface for fetching linked markdown.
The built-in loaders support both local files and HTTP resources. Path resolution
includes security checks that block directory traversal and sensitive system paths.

> Things to remember:
>
> - First item
> - Second item
> - Third item

Style definitions are organized into named themes: dark, light, ascii, dracula,
tokyo-night, and pink. Each theme configures colors, indentation, prefixes, and
code block syntax highlighting through Chroma.

## Multi-paragraph blockquote

Navigation history uses a generic stack with back and forward operations. Each
page state captures a full snapshot including scroll position, selected element,
and rendered content, enabling seamless back/forward navigation.

> First paragraph of the blockquote. This has enough text to potentially
> wrap across multiple lines depending on the terminal width.
>
> Second paragraph, still inside the same blockquote block.

The word wrap setting controls how long lines are broken. When set to zero,
lines are not wrapped and extend to their natural length. The margin writer
handles indentation and padding around block elements like blockquotes and
code blocks.
