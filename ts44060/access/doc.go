// Package access contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §11.2.2; access bursts retain their printed selectors and exact significant extent. Application procedures remain external.
// TS 44.060 V19.0.0 §11.2.5; access bursts retain their printed selectors and exact significant extent. Application procedures remain external.
// TS 44.060 V19.0.0 §11.2.5a tables 11.2.5a.2 and 11.2.5a.4: non-PEO requests are eleven bits at origin zero. PEO and MTA body components are eight bits at origin three after the table-owned selector. The extended-method part-2 component covers bits 4–11 only. The complete thirty-bit extended part 2 is additionally selected from table 11.2.5a.4 row 5 and table 11.2.5a.4.1; its context-qualified codec keeps the original eight-bit body API unchanged.
// TS 44.060 V19.0.0 §11.2.5c; access bursts retain their printed selectors and exact significant extent. Application procedures remain external.
//
// TS 44.060 V19.0.0 §11.2.44; access bursts retain their printed selectors and exact significant extent. Application procedures remain external.
package access
