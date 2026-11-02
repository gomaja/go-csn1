// The VGCSNeighbourCellInformation codec accepts every printed three-bit
// Validity bitmap, including zero, and preserves the received bits on replay.
// TS 44.018 V19.0.0 §9.1.57 p11507 requires a receiving mobile station to
// diagnose an imperative message part error for an all-zero bitmap. That
// application procedure belongs to the caller; decoding does not perform it.
package messages
