# IDL Syntax

<!-- toc -->
- [Design Goals](#design-goals)
- [Examples](#examples)
- [Design Notes](#design-notes)
  - [Uhhh, where are the proto tags?](#uhhh-where-are-the-proto-tags)
  - [What's up with markers?](#whats-up-with-markers)
- [Unresolved/Open Issues](#unresolvedopen-issues)
- [Formal(-ish) Grammar](#formal-ish-grammar)
<!-- /toc -->

## Design Goals

1. High-level concepts and primitives in kubernetes should have
   corresponding concepts in the language.  For instance:

   * Quantity is fundamentally a primitive in Kubernetes, and thus should
     have a corresponding primitive in our IDL.

   * Kind is a fundamental concept in Kubernetes, so it should just be
     possible to declare a kind in the IDL.

2. Kubernetes API requirements should be encoded in the language:
   disallowed constructs (and many bad practices) in Kubernetes APIs
   should not be possible to represent in the IDL.

3. Kubernetes API recommendations should be the default behavior in the
   language: when not explicitly overridden, users should get our
   recommended behavior by default:

   * For example, the recommendation is new list-maps have a merge key of
     name.  When merge keys aren’t manually specified, this is the default
     one.

4. All information conveyed should be as structured as possible

5. Naming in the IDL describes the serialized form.  Language bindings
   (Go) are derived automatically from that.  Specific overrides for
   legacy codebases occur via markers, since they are exceptions to the
   rules.

## Examples

***Author's note***: the cannonical source for these examples is [in the
repository][repo-example]; the example reproduced here is just to make it
easier to comment.

[repo-example]: https://github.com/DirectXMan12/idl/blob/main/kdlc/testdata/all-around.kdl

```
// double slashes add comments
/* same with block comments */

// Triple-slash comments denote documentation
// If a line contains `# Words`, the following lines
// will be placed in the corresponding OpenAPI documentation
// field.

/// core/v1 describes core k8s concepts
group-version(group:"core", version:"v1") {
    /// Pod is a group of related, colocated processes
    ///
    /// # Example
    /// {"apiVersion": "v1", "kind": "Pod", "spec": {}}
    kind Pod {
        // types may be nested -- this is just a convenience for namespacing
        // it does not imply privacy or anything

        spec: Spec,
        struct Spec {
            // list-maps are ordered maps whose keys appear in their body.
            // not specifying a key defaults to `key: name`.
            // Go represents them as lists with specific merge keys, other
            // languages may represent them other ways.
            // They are automatically merged as maps in SMD.

            // optional fields must list the keyword `optional` before the
            // type.

            /// list of volumes that can be mounted by containers belonging to
            /// the pod
            volumes: optional list-map(value: Volume),

            // ...

            dnsPolicy: optional(default: ClusterFirst) DNSPolicy,

            // simple-maps are unordered maps from string-type values to
            // primitive values
            nodeSelector: optional simple-map(value: string),

            // ...

            @deprecated(msg: "use serviceAccountName instead")
            serviceAccount: optional string,

            // ...

            readinessGates: optional list(value: ReadinessGate),
            struct ReadinessGate { }

            // create-only fields are immutable after creation (see immutability KEP)
            containers: create-only list-map(value: Container),
            struct Container {
                name: string,
                // ...
            }
        }

        status: Status,

        struct Status {

        }
    }

    wrapper DNSLabel: string validates(pattern: "[a-z0-9]([-a-z0-9]*[a-z0-9])?", max-length: 64);

    struct Volume {
        name: DNSLabel,
        source: Source,
    }

    // enums represent one choice of a series of constant string values
    enum DNSPolicy {
        /// resolve cluster first, without host networking
        ClusterFirstWithHostNet,
        /// resolve cluster first
        ClusterFirst,
        /// resolve in the default order
        Default,
        /// don't resolve DNS
        None,
    }

    // markers are extensions that annotate fields or types with
    // additional data that can be picked up by tooling.
    // several markers are built-in (e.g. `deprecated`)
    //
    // The examples below imagine a marker that denotes that a given
    // field actually contains a reference to another object.
    // Other markers might include overrides for legacy Go type names,
    // marking fields as being behind feature gates for automated linting
    // and better documentation, etc.

    struct SecretProjection {
        // inline fields have their fields embedded in their parent object
        // in systems where this is supported (e.g. JSON, Go).  In systems
        // where not supported, generally just use the type name as the field
        // name.  Only structs may be inlined.

        // "raw identifiers" are surrounded with backticks, and take two purposes
        // the first is to allow fields and keys that have the same name as keywords
        // (like below).
        //
        // The second is as an emergency "break glass" to violate the "field name must
        // have an lowercase first letter" constraint.
        @reference(group: "v1", version: "core", `kind`: "Secret")
        _inline: LocalObjectReference,
    }

    @reference(same-namespace: true)
    struct LocalObjectReference {
        // fields are referenced using `.name`
        // (this avoids parsing ambiguities with primitive type names)

        @reference-part(part: .name)
        name: string,
    }

}
```

```
import (
    types (
       // group-versions are imported explicitly from KDL or CKDL files
       {core/v1} from "k8s.io/api/core/v1";
    )
    markers (
        // marker definitions are imported from special KDL or CKDL files
        // and given a prefix whereever they're used.
        kgo from "kubernetes.mkdl";
    )
)

group-version(group: "examples", version: "v1") {
    kind CSIDriver {
        spec: Spec,

        struct Spec {
            // ...

            // in a set, each item may appear only once.
            volumeLifecycleModes: optional set(value: VolumeLifecycleMode),
        }
    }

    kind StorageClass {
        // ...

        // types from other group-versions may be referenced with
        // `group/version::path`.
        reclaimPolicy: optional(default: Delete) core/v1::PersistentVolumeSpec::ReclaimPolicy,

        @kgo::feature-gated(name: "VolumeScheduling")
        volumeBindingMode: optional VolumeBindingMode,
    }
}
```

## Design Notes

### Uhhh, where are the proto tags?

Earlier versions of this proposal included a specific syntax for adding
proto tags (`myField[42]: optional string` introduces a field named
`myField` with a tag of `42`). However, this syntax was not entirely
obvious.

The current plan is to place proto tags in a look-aside file that will
only ever be appended to, and will be managed by the compiler.  This has
the *slight* downside of requiring extra scrutiny on field renames, but
has the upside that users *don't have to care* or learn about how proto
tags work, why we'd have to leave room for reserved fields in kinds, etc.

The tags will still be present in the CKDL file, meaning that tooling has
proto tags available for use all from a single file, while humans don't
have to bother with them.

### What's up with markers?

Markers are intended to be the extension mechanism for the language. 
They’re intended to allow “marking” (a.k.a. annotating) fields and types
with tooling-specific details.

The general pattern is that things involved in the standard serialization
pipeline are built into the language: JSON-focused field naming, type data
(types, optionality, extra constraints like validation, etc).

These represent the “happy path” of details that all API authors (builtin,
CRD) need to care about, as well as details that most (if not all)
serialization & generation pipelines will need to consider (practically,
they have first-class representations in CKDL well).

On the other hand, things that are specific to a particular code
generation pipeline, or that add additional informational data intended to
be picked up by certain linters, etc, will generally be markers.

**So why not just make everything a marker?** This comes down to visual
interpretation: separating out type data from additional data makes it
easier to visually scan for details, review, etc: you know where to look
for what data.

This plays into the positioning of markers as well -- markers go above,
type modifiers go inline.  This is mirrored across most languages with
such an extension point.

From a serialized perspective, it makes it easier for folks writing code
that consumes CKDL to know what they must deal with.  If everything is an
extension in the serialized CKDL, it’s hard to look and know what the
structure of the serialized form is.

*That being said*, syntactically, markers and “type modifiers” (everything
after the colon) use the same syntax (`key ~ named_param_list?`) modulo
the `@`, so it’s possible that we could make that positioning an
"either-or" -- `@foo(...)` above a field, or just `foo(...)` inline. 
However, this makes it harder to know where to look for certain bits of
data when reading KDL, so it’s not clear to me what the benefit is beyond
a conceptual linguistic equivalence.

## Unresolved/Open Issues

<<[UNRESOLVED @directxman12]>>
Earlier versions of this proposal involved specifying marker definitions
using proto files, since they compile down to proto messages.  It was
pointed out that this is a bit hacky, so the proposal was switched to
allow defining markers via KDL instead (and just translating that to the
proto definitions).  Exact syntax is still tbd, but I'd imagine it'd look
like:

```
marker deprecated {
    msg: string,
}
```
<<[/UNRESOLVED]>>

## Formal(-ish) Grammar

***Author's note***: the canonical source for this grammar is [in the
repository][repo-grammar]; the version reproduced here is just to make it
easier to comment.

[repo-grammar]: https://github.com/DirectXMan12/idl/blob/main/docs/grammar.txt

```
// a note on syntax for this file: the syntax for this file
// roughly follows the [pest](https://pest.rs) syntax, for
// various historical reasons & since there's no single standard
// for ebnf.
//
// TL;DR:
// - `~` means adjacent, but can include whitespace or comments
// - `*`, `+`, `?`, `|`, and parentheses have their usual meanings
// - `@` before the braces of a rule indicates that no whitespace
//   or comments may be present between tokens unless explicitly
//   noted (i.e. `~` means *immediately adjacent*).
// - `!` means look-ahead negation, so `!NEWLINE ~ ANY` means any
//   character that's not a newline.

file = { SOI ~ imports? ~ qualified_decl+ ~ group_version+ ~ EOI }

// imports may either be types or marker (see below)
// defintions
imports = { (import_one | import_both) }

// there's a short form for when you only need one in a file
import_one = { "import" ~ (import_types | import_markers) }
import_both = { "import" ~ "(" ~ import_types ~ import_markers ~ ")" }

// all types fall under a group-version.  Multiple
// group-versions may be specified per file, and we can mark
// that we need specific group-versions from a file.
//
// Explicitly calling out the group-versions that we need
// makes it a *lot* more obvious what comes from where.
// Types are imported from kdl files (source or compiled),
// whose paths are specified via the given string.
import_types = { "types" ~ "(" ~ type_import* ~ ")" }
type_import = { "{" ~ (group_version_ident ~ "," )* "}" ~ "from" ~ string ~ ";" }

// markers are imported by giving them an alias prefix (much
// like `import ( alias "pkg" )` in Go).
import_markers = { "markers" ~ "(" ~ marker_import* ~ ")" }
marker_import = { key ~ "from" ~ string ~ ";" }

group_version_ident = @{ group_name ~ "/" ~ version_name }

// Grammar note: no extra newlines or additional comments may occur between
// docs, markers, and the things that they describe.  This is not directly
// expressed in the grammar below to make it more readable.

// Grammar note: in many cases, named_param_lists will have an effective set of
// allowed values.  These will be noted in the comments like `PARAMS(param: type, param: type)`.

// qualified types are a shorthand for specifying a type in a
// group-version in a non-nested form.  This may be useful for
// very long files where a periodic reminder is useful, or if
// you want a single file with one type across several group-versions,
// and don't want to repeat the nesting.
//
// For example,
//
// ```
// group-version(group: "core", version: "v1") {}
//
// kind core/v1::Pod {}
// enum core/v1::ConditionStatus {}
// ```
//
// is equivalent to
//
// ```
// group-version(group: "core", version: "v1") {
//    kind Pod {}
//    enum ConditionStatus {}
// }
// ```
//
// The equivalent group-versions must still be specified elsewhere in the file
// even if they are empty.  You can use the empty group-version declarations
// to attach markers and documentation to the group-version.
qualified_decl = { doc? ~ markers? ~ (qualified_kind_decl | qualified_struct_decl | qualified_enum_decl | qualified_wrapper_decl) }

// PARAMS(group: string, version: string)
group_version = { doc? ~ markers? ~ "group-version" ~ named_param_list ~ "{" ~ decl+ ~ "}" }

// documentation takes the form of syntactically distinct
// comments (they start with `///`, as in TypeScript, Rust,
// C#, etc, to easily distinguish them from normal comments
// and avoid mistakes).
doc = @{ ((" " | "\t")* ~ (doc_empty | doc_content | doc_section))+ }
doc_empty = @{ "///" ~ NEWLINE }
doc_content = @{ "/// " ~  ((!(NEWLINE | "#") ~ ANY) ~ (!NEWLINE ~ ANY)*)? ~ NEWLINE }
// documentation lines that start with `#` indicate a
// documentation section.   The supported sections are
// the default one ("description"), `Example`, and `External
// Reference` (for external links to more documentation).
// They roughly correspond to OpenAPI fields.
//
// For example
//
// ```
// This is the description, and gets put in "description"
// # Example
// {"this": "stuff goes in the example field"}
// ```
doc_section = @{ "/// #" ~ (!NEWLINE ~ ANY)+ ~ NEWLINE }

// all "parameters" given to types, markers, type-modifiers,
// etc take the form of lists of key-value pairs.  All
// arguments *must* have a keyword (no positional arguments)
// -- this makes future modifications, changing of behavior,
// etc much easier, and makes it much more obvious what a
// given parameter means.
named_param_list = { "(" ~ key_value ~ ("," ~ key_value)* ~ ")" }

// Parsing note: keys & certain field names have a syntax that appears to
// overlap with keywords.  This is not allowed -- key & field_ident implicitly
// exclude keywords, and raw keys/raw identifiers may be used to get around this.
// It's possible that we could relax this rule in the future, but we'll err on
// the side of caution for now.

key_value = { (key | raw_key) ~ ":" ~ value }
// raw_keys are used as an escape hatch when a key would conflict with a keyword.
raw_key = @{ "`" ~ key ~ "`" }
key = @{ LOWERCASE_LETTER ~ (LOWERCASE_LETTER | "-" | DECIMAL_NUMBER)+ }

// allowed values are a slight superset of JSON.  In
// addition to JSON, we define 2 new value types to make
// structured type-checking easier: types and fieldPaths.
// we also allow "key" values where string keys would be
// allowed in JSON.

// Parsing note: trailing commas are always optional,
// but expressing that in a grammar makes it less readable.

value = { number | string | bool | struct_val | list_val | type_mod | field_path }
struct_val = { "{" ~ (struct_kv ~ ",")* ~ "}" }
struct_kv = { (key | raw_key | string) ~ ":" ~ value }
list_val = { "[" ~ (value ~ ",")* ~ "]" }
// field paths specify a particular field in an object,
// which the type-checker can then confirm exists.  This is
// mainly useful for ensuring that you don't typo things
// like list-map key names, or certain markers.
field_path = @{ "." ~ field_identifier }

// declarations are either "kinds" or some sub-type that may
// be referenced in a kind.
//
// struct-like declarations (kinds, struct-subtypes)
// may have nested declarations.  This
// provides some automatic namespacing, and makes it
// possible to place single-use types adjacent to their
// usage (no more skipping back and forth between definition
// and usage in a long file).
decl = { doc? ~ markers? ~ (kind_decl | struct_decl | enum_decl | wrapper_decl) }

// Type identifiers are PascalCase (UpperCamelCase,
// `LikeThis`) to distinguish them from field identifiers,
// which are camelCase (lower-case first letter).
// This matches the Kubernetes API conventions.
type_identifier = @{ UPPERCASE_LETTER ~ ( LETTER | DECIMAL_NUMBER )+ }

// kinds are identical to structs syntactically, but have
// additional semantics: they're fundamentally "kinds" in
// kubernetes, and thus automatically have typemeta,
// corresponding list types, etc where appropriate.
//
// By default, kinds are considered to be "persisted", and
// thus also have objectmeta (turned off by setting
// `nonpersisted`).
// PARAMS(nonpersisted: bool)
kind_decl = { "kind" ~ named_param_list? ~ type_identifier ~ "{" ~ struct_body ~ "}" }
qualified_kind_decl = { "kind" ~ named_param_list? ~ qualified_type_ref ~ "{" ~ struct_body ~ "}" }

struct_decl = { "struct" ~ type_identifier ~ "{" ~ struct_body ~ "}" }
qualified_struct_decl = { "struct" ~ qualified_type_ref ~ "{" ~ struct_body ~ "}" }
struct_body = { (field ~ ",") | decl)* }
// a field is specified by identifying information before
// the colon (name, proto tag), and type & validation
// information after.
//
// Field names are `camelCase` (lower-case first letter),
// directly equivalent to their JSON forms.
//
// Inline fields use `_` instead of a field name
field = { doc? ~ markers? ~ (field_identifier | raw_field_identifier | "_inline") ~ ":" ~ type_spec }
// raw_field_identifiers avoid conflicts with keywords (like raw_key) and also provide an escape
// hatch for violating field naming conventions for certain legacy cases.  By having a separate
// syntax, we explicitly call out that this is strange and not supposed to be the common case.
//
// Namely: uppercase-first-letter is necessary for one mistake in core/v1,
// and shishkebab-case (`like-this`) is necessary for the clientcmd
// (kubeconfig) API.
raw_field_identifier = @{ "`" ~ LETTER ~ ( LETTER | DECIMAL_NUMBER | "-")+ ~ "`" }
field_identifier = @{ LOWERCASE_LETTER ~ ( LETTER | DECIMAL_NUMBER )+ }

// enums are string-valued enumerations (think
// ConditionStatus or ReclaimPolicy).  Each value is
// specified as a TypeIdentifier and is serialized literally
// in JSON, as per our API conventions.
enum_decl = { "enum" ~ type_identifier ~ "{" ~ ((type_identifier ~ ",")* ~ "}") }
qualified_enum_decl = { "enum" ~ qualified_type_ref ~ "{" ~ ((type_identifier ~ ",")* ~ "}") }

// wrappers are aliases to other types with their own
// semantics (e.g. extra validation, or some inherent
// meaning).  Their syntax is intended to be reminiscent of
// field declarations.
wrapper_decl = { "wrapper" ~ type_identifier ~ ":" ~ type_spec ~ ";" }
qualified_wrapper_decl = { "wrapper" ~ qualified_type_ref ~ ":" ~ type_spec ~ ";" }

// a type-spec is a list of "type modifiers".  It must eventually
// include exactly one concrete type, but this does not
// have a bearing on the actual grammar.
type_spec = { type_mod+ } // must include act_type at some point
type_mod = { (key ~ named_param_list?) | type_ref }

// valid modifiers include:
// - primitives: string, int32, int64, quantity, time, duration, bytes, bool, int-or-string, dangerous-float64
// - collection types: set(value: type), list-map(value: type, keys: list-of-types), list(value: type), simple-map(key: type, value: type)
// - References: bare type identifiers, local type references, qualified type references
// - behavior modifiers: optional(default: value), create-only, validates(...), preserves-unknown-fields, embedded-kind

// lists are ordered atomic collections of types:
//
// list(value: type_mod)

// sets are sets of items
//
// set(value: type_mod)

// list-maps are ordered maps of items that serialize as lists
//
// list-map(value: type, key: list-of-field-paths)

// simple-maps are unordered maps.  They're eventually restricted
// to string-equivalent keys and largely primitive values, as per
// the Kubernetes API conventions
//
// They are largely used for label sets, selectors,
// and resource-list.
//
// We don't just call this "map" so that we make it clear
// that k8s has two types of maps: one for primitives,
// and another for compound types.  This is a confusing
// point of the k8s API guidelines, so it's worth making
// this more explicit in the IDL.
//
// simple-map(key: type, value: type)

// the optional modifier is used to mark a field as optional,
// or as "optional-but-defaulted" (in which case the field
// appears as optional on writes, but will always be populated
// with a value by the API server if one is not submitted)
//
// optional
// optional(default: value)

// the validates modifier is used to apply extra validation.
// For the moment, it has keys equivalent to nearly all OpenAPI
// fields, with the exception of the combinator fields and the
// format field, which is represented by the primitives.
//
// validates(key: any, key: any, ...)

// type references may be local or fully-qualified.
//
// Local references are resolved by first looking in the
// parent declaration, then proceeding up till the
// containing group-version).
type_ref = { type_identifier | qualified_type_ref }
// qualified types are *always* first qualified with
// group-version (same syntax used for imports, specifying
// apiVersion, etc), followed by a `::`-separated list of
// nested types.
qualified_type_ref = ${ group_ver ~ "::" ~ type_identifier ~ ("::" ~ type_identifier)* }

// markers are the extension point for KDL.  They're
// intended to be used to attach information specific to a
// particular tool (e.g. k/k's go-gen, proto-gen, etc) or to
// experiment with attaching additional semantics to the
// language.
markers = { marker* }
marker = { "@" ~ key ~ named_param_list? }

// normal comments are either single-line or multiline, just
// like Go/C/Rust/etc
line_comment = @{!"///" ~ "//" ~ (!NEWLINE ~ ANY)* ~ NEWLINE }
inline_comment = @{ "/*" ~ (!"*/" ~ ANY)* ~ "*/" }

// these follow the JSON spec (except strings must be valid utf-8).
// number explicitly excludes decimals, which do not exist in Kubernetes.
number = @{ "-"? ~ non_zero_digit ~ ASCII_DIGIT* }
string = @{ "\"" ~ str_char* ~ "\"" }
str_char = @{ (!(NEWLINE | "\"" | "\\") ~ ANY) | ("\\" ~ escape) }
escape = @{ "\"" | "\\" | "/" | "b" | "f" | "n" | "r" | "t" | ("u" ~ ASCII_HEX_DIGIT ~ ASCII_HEX_DIGIT ~ ASCII_HEX_DIGIT ~ ASCII_HEX_DIGIT) }
non_zero_digit = @{ !"0" ~ ASCII_DIGIT }
bool = { "true" | "false" }

// bare group & version names have identical semantics to
// their requirements in Kubernetes: groups are DNS
// subdomains...
group_name = @{ dns_label ~ ("." ~ dns_label)* } // DNS subdomain (RFC 1123)
// ...and versions follow the structured forms outlined in the
// CRD version sorting algorithm
version_name = @{ "v" ~ non_zero_digit ~ ASCII_DIGIT* ~ (("alpha"|"beta") ~ non_zero_digit ~ ASCII_DIGIT*)? }
dns_label = @{ ASCII_ALPHA_LOWER ~ (ASCII_ALPHA_LOWER|"-"|ASCII_DIGIT){1,62} }

WHITESPACE = _{" " | "\t" | "\r" | "\n" }
COMMENT = @{ line_comment | inline_comment }
KEYWORDS = { "true" | "false" | "group-version" | "kind" | "struct" | "enum" | "wrapper" | "import" | "types" | "markers" }
```
