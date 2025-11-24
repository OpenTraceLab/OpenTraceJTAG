# Gio Affine Transform Color Bug Demo

## Issue Description

When using `op.Affine` to apply a scale transform, colors render noticeably paler/lighter compared to untransformed content. This affects all shapes (rectangles, circles) rendered with `paint.FillShape`.

## Reproduction

1. Build and run:
   ```bash
   go mod download
   go build
   ./gio-transform-bug
   ```

2. Observe:
   - Left side: Shapes without transform (reference colors)
   - Right side: Shapes with current zoom transform
   - Press `+` to zoom in - colors get paler
   - Press `-` to zoom out - colors get paler
   - Press `R` to reset to zoom=1.0 - colors return to normal

## Expected Behavior

Colors should remain consistent regardless of transform scale. The same `color.NRGBA` values should produce the same visual appearance whether rendered with or without an Affine transform.

## Actual Behavior

Colors become noticeably paler when any scale transform is applied (zoom != 1.0). The effect is binary - either colors are correct (zoom=1.0) or pale (zoom!=1.0), not gradual.

## Environment

- Gio version: v0.9.0
- OS: Linux (WSL2/X11), but may affect other platforms
- Go version: 1.21+

## Workaround

Skip the transform entirely when zoom is exactly 1.0:

```go
if zoom != 1.0 {
    defer op.Affine(f32.Affine2D{}.
        Scale(f32.Point{}, f32.Pt(zoom, zoom))).Push(gtx.Ops).Pop()
}
// render content
```

## Additional Notes

This issue also affects text rendering (fonts appear paler when scaled). It appears to be related to GPU/driver anti-aliasing or color blending behavior when rendering transformed content.
