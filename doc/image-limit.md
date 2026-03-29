# Kitty Image Storage Quota and Downscaling

## Problem

Images in markdown documents with many large diagrams would intermittently
disappear. The issue was deterministic based on total image pixel volume:
whichever images were transmitted first would be silently evicted once the
quota was exceeded.

## Root Cause

Kitty terminal maintains a **320MB storage quota** for decoded image data
(uncompressed RGBA in GPU texture memory). When a new image transmission
pushes total usage over the limit, Kitty silently evicts the least recently
used image. No error is sent to the client.

We were transmitting images at their **full source resolution** and relying
on Kitty's `c=<cols>,r=<rows>` parameters to scale on the GPU. But Kitty
decodes the full image into RGBA memory before scaling. A single 6124x5470
mermaid diagram consumed 128MB of the 320MB quota. With 5+ large diagrams,
the quota was exceeded and the oldest images disappeared.

## Solution

Downscale images to the **actual display pixel dimensions** before
transmission. The target size is `cols * cellWidth` by `rows * cellHeight`
— the exact pixel area the image occupies on screen. This is done in
`ImageManager.EnsureTransmitted()` using the existing `ImageInfo.ResizedPNG()`
method.

A 6124x5470 image displayed in an 80-column terminal (8px cells) now
transmits as ~640x570 pixels (~1.4MB RGBA) instead of the original 128MB.
This is an ~80x reduction in Kitty memory usage per image with no visible
quality loss, since the terminal can't display more detail than the cell
grid allows.
