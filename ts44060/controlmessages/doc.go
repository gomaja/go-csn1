// Package controlmessages contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §11.2.8; exact printed content root, with externally supplied bit extent.
// These codecs require the caller to supply the exact content extent.
// MAC framing and message selectors remain with the enclosing control block.
//
// TS 44.060 V19.0.0 §11.2.30a, printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.35, printed content and externally supplied extent.
//
// TS 44.060 V19.0.0 §11.2.1 paragraphs 13464–13491. The caller supplies the containing message extent; selecting an indefinite Ignore preserves the accepted typed prefix and ends the enclosing repeated and nested descriptions. Fresh and canonical senders select semantic arms.
// TS 44.060 V19.0.0 §11.2.11, paragraphs 16345–16349; message content at the caller-supplied original block bit coordinate and extent.
// TS 44.060 V19.0.0 §11.2.14, paragraphs 16444–16448; message content at the caller-supplied original block bit coordinate and extent.
// TS 44.060 V19.0.0 §11.2.15, paragraphs 16460–16467; message content at the caller-supplied original block bit coordinate and extent.
// TS 44.060 V19.0.0 §11.2.2a, paragraphs 13579–13588; message content at the caller-supplied original block bit coordinate and extent.
// TS 44.060 V19.0.0 §11.2.8b; exact printed content root, with externally supplied bit extent.
//
// TS 44.060 V19.0.0 §11.2.12, paragraphs 16363–16374; message content at the caller-supplied original block bit coordinate and extent.
// TS 44.060 V19.0.0 §11.2.17, printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.9c, paragraphs 16049–16056; message content at the caller-supplied original block bit coordinate and extent.
//
// TS 44.060 V19.0.0 §11.2.13, authenticated printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.3, authenticated printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.32, authenticated printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.34, authenticated printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.3a, authenticated printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.45, authenticated printed content and externally supplied extent.
// TS 44.060 V19.0.0 §11.2.5b, authenticated printed content and externally supplied extent.
package controlmessages
