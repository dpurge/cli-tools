#let _baseFont = (
  "Gentium", "Charis SIL", "Noto Serif",
  "Amiri", "Scheherazade", "Noto Naskh Arabic", "Noto Sans Arabic",
  "Ezra SIL", "Frank Ruehl CLM", "Noto Serif Hebrew", "Noto Sans Hebrew",
  "AR PL UMing", "AR PL UKai",
  "Baekmuk Batang",
  "Noto Sans",
)

#let _roleFonts = state("role-fonts", (
  body: _baseFont, header: _baseFont, transcription: _baseFont, translation: _baseFont, strong: _baseFont, emph: _baseFont, notes: _baseFont,
))

#let _sourceDir = state("source-dir", ltr)

// SPECS §8.2/§8.3: the full parsed font.css slot table (script/extension/
// field/style-qualified families, keyed by the SAME canonical join
// pkg/ebook's FontTable uses), populated once in book(). #_resolveFont
// below is the Typst-side mirror of FontTable.resolve (typst.go) — it MUST
// stay algorithmically identical to it (ASR-4/Major-2), since only this
// template sees each block's own script/extension/field at compile time.
#let _fontSlots = state("font-slots", (:))

// book-script is passed through for forward-compat/debugging ONLY (SPECS
// A2: resolution keys exclusively on each block's OWN resolved script,
// never the book's) — #_resolveFont deliberately never reads this state.
#let _bookScript = state("book-script", "")

// largeScriptCodes mirror (typst.go's largeScriptCodes, kept in sync by
// hand — a small, rarely-changing closed set): scripts whose synthetic
// bold/italic renders poorly, so strong/emph substitute a distinct role
// font at normal weight/style instead (SPECS §8.4).
#let _largeScripts = (
  "hans", "hant", "hani", "arab", "hebr", "kore", "hang", "jpan", "hira", "kana", "syrc",
)
// Defensively lowercase/trim before the set-membership check, matching the
// Go-side `resolve`'s `strings.ToLower(strings.TrimSpace(...))` normalization
// (SPECS §8.4 FIX-4/minor): a caller passing e.g. "Arab" (author-cased
// script= attribute) must gate identically to "arab".
#let _isLargeScript(script) = lower(script.trim()) in _largeScripts

// SPECS F-SIZE: per-block complex-script enlargement. Rather than the book-
// level uniform bump (book()'s `size` below, keyed on the BOOK script), each
// block enlarges ONLY its own foreign-script run so a Latin-script book with
// e.g. hans blocks (script:latn book, script=hans blocks) renders the Chinese
// larger than the surrounding pinyin/translation. `_sizeFactor` is set in
// book(): 1.0 when the book is already whole-book large-script (no double
// enlargement / no regression), else size-large/size (≈1.33). Reads state, so
// callers MUST be inside a `context` expression (mirrors _resolveFont, above).
#let _sizeFactor = state("size-factor", 1.0)
// _foreignSizeFactor returns the bare numeric enlargement factor for a script
// (reads _sizeFactor state, so callers MUST be inside a `context` expression).
// Used by _foreignSize AND by the FR-1 heading counter (textblock else branch)
// to divide out the ambient enlargement from headings.
#let _foreignSizeFactor(script) = if _isLargeScript(script) { _sizeFactor.get() } else { 1.0 }
#let _foreignSize(script) = _foreignSizeFactor(script) * 1em
// _baseSizeFactor holds the ratio that returns the document-ambient text size
// to the book's base `size` when the book is a large-script book (where the
// ambient is size-large). Set by book() to size/size-large when large-script is
// true, else 1.0 (no-op). Reads state, so callers MUST be inside a `context`
// expression (same convention as _foreignSizeFactor above).
#let _baseSizeFactor = state("base-size-factor", 1.0)
#let _baseSize() = _baseSizeFactor.get() * 1em

// SPECS F-MARK: the content-type badge — a filled black square with a knockout
// white letter (T/V/D/M/Q). The letter's font is PINNED to the header (Latin)
// role so it never inherits a surrounding CJK/complex-script stack and turn
// into a substitution box (reviewer H2); reading _roleFonts needs `context`.
#let _ctbadge(letter) = context box(
  width: 1.1em, height: 1.1em, fill: black, radius: 1pt, inset: 0pt, baseline: 0.2em,
  align(center + horizon, text(font: _roleFonts.get().header, fill: white, weight: "bold", size: 0.72em, letter)),
)

// _slotKey joins the non-empty (script,ext,field,style) parts with a single
// space — the SAME canonical key shape typst.go's slotKey produces, so a
// family classified there and looked up here always agree.
#let _slotKey(..parts) = parts.pos().filter(p => p != "").join(" ")

// _baseRoleFor mirrors typst.go's baseRoleForField (SPECS §4's "BaseRole(F)
// map"): Source/Content/Main/Question/Answer/Phrase -> Body (or Translation
// when as-translation); Transcription -> Transcription; Translation/
// Secondary/Grammar -> Translation; Tag/Header -> Header.
#let _baseRoleFor(field, as-translation) = {
  if field in ("source", "content", "main", "question", "answer", "phrase") {
    if as-translation { "translation" } else { "body" }
  } else if field == "transcription" {
    "transcription"
  } else if field in ("translation", "secondary", "grammar") {
    "translation"
  } else if field in ("tag", "header") {
    "header"
  } else {
    "body"
  }
}

// _roleFontsKey maps a style name to _roleFonts' pre-existing dict field
// name: the legacy state uses "emph", not "emphasis" (ASR-1, unchanged).
#let _roleFontsKey(style) = if style == "emphasis" { "emph" } else { style }

// _resolveFont implements SPECS §4's field -> extension -> script ->
// base-role chain (mirrors FontTable.resolve, pkg/ebook/typst.go). It
// returns a font ARRAY ready for `text(font: ...)`: the most-specific
// declared family (if any) followed by the legacy base-role's already-
// assembled stack (font-<role> arg + the base multi-script font, i.e.
// exactly what _roleFonts.get() already carries) — so an undeclared
// qualified slot falls through to today's behavior with no visible change
// (ASR-1). Must be called from within `context` (reads state).
//
// style selects the Strong/Emphasis sub-axis ("strong"/"emphasis"; "" =
// regular). as-translation mirrors {start-text as=translation}: primary
// fields resolve base role Translation instead of Body. An empty script
// SKIPS the script-qualified levels (SPECS §6: no book-Script inheritance,
// G1 deferred) and resolves via the base role directly — this is also how
// callers realize Major-2's fixed-script fields (pass script: "latn" for
// transcription, script: "" for translation/grammar/secondary, regardless
// of the block's own foreign `script` param).
#let _resolveFont(script: "", ext: "", field: "", style: "", as-translation: false) = {
  // Defensively lowercase/trim every axis before building candidate keys
  // (SPECS §8.4 FIX-4/minor), matching the Go-side FontTable.resolve
  // normalization — so an author-cased attribute (e.g. script="Arab") or a
  // caller passing a not-yet-normalized field/style still joins the SAME
  // canonical _slotKey a differently-cased classifyFontFamily entry would.
  let script = lower(script.trim())
  let ext = lower(ext.trim())
  let field = lower(field.trim())
  let style = lower(style.trim())
  let slots = _fontSlots.get()
  let base-role = _baseRoleFor(field, as-translation)

  if style != "" {
    let candidates = ()
    if script != "" {
      candidates = (
        _slotKey(script, ext, field, style),
        _slotKey(script, ext, style),
        _slotKey(script, style),
      )
    }
    candidates.push(style)
    for c in candidates {
      if c in slots {
        return (slots.at(c),) + _roleFonts.get().at(_roleFontsKey(style))
      }
    }
    // None declared: fall through to the regular (unstyled) chain below —
    // book.typ's per-block gate (§8.4) already decided the regular font is
    // an acceptable substitute here.
  }

  let candidates = ()
  if script != "" {
    candidates = (
      _slotKey(script, ext, field),
      _slotKey(script, ext),
      _slotKey(script),
    )
  }
  for c in candidates {
    if c in slots {
      return (slots.at(c),) + _roleFonts.get().at(base-role)
    }
  }
  _roleFonts.get().at(base-role)
}

// Structured block helpers (SPECS §7.2 / §8.2). _blockheading emits a
// Typst heading at the given level, excluded from the document outline
// (outlined: false, D4 / ASR-6) so block headers never appear in the TOC
// but still inherit book()'s per-level heading show-rules (font/size).
// _blocknote emits centred text in the notes role font (ASR-5 fallback:
// emphFont when font.css declares no Font Notes). No large-script gate:
// text(font:) is called directly, bypassing book()'s strong/emph
// interceptors (SPECS §7.2, D3).
// FR-5: center all structured-block headers (dialog/vocabulary/models/questions).
// FR-6 (start-dialog): outlined: false already keeps these out of the TOC;
// the align() wrapper does not disturb the outlined field (verified by compile).
#let _blockheading(level, body) = align(center, heading(level: level, outlined: false)[#body])
#let _blocknote(body) = align(center, context text(font: _roleFonts.get().notes)[#body])

#let textblock(role: "source", dir: ltr, script: "", body) = {
  // FR-6: exclude ALL headings inside a start-text block from the outline(),
  // mirroring _blockheading's outlined: false (start-dialog, etc.). Raw markdown
  // headings inside textblock() emit plain Typst `= ...` syntax, which defaults
  // to outlined: true; this rule overrides that for every heading in this scope.
  show heading: set heading(outlined: false)
  show heading.where(level: 1): set align(center)
  show heading.where(level: 2): set align(center)
  show heading.where(level: 3): set align(center)
  set text(dir: dir)

  // SPECS Major-2: transcription/translation/grammar resolve their FAMILY
  // with their own fixed script (transcription -> latn, translation/
  // grammar -> base ""), decoupled from this block's foreign `script`
  // param, so a bare `Font <Script>` catch-all in font.css can't hijack
  // e.g. a Polish translation column inside an Arabic block.
  let familyScript = if role == "transcription" { "latn" } else if role == "translation" or role == "grammar" { "" } else { script }

  // SPECS §8.4 (review-amended 2026-07-31): the Strong/Emphasis GATE MUST use
  // the SAME script as this field's family resolution (familyScript), NOT the
  // raw block `script`. Using the raw script here was the code-review defect:
  // for role=translation/grammar, familyScript is fixed ("" / base) but the
  // raw `script` is still the block's own foreign script — gating on the raw
  // script would re-enter the large-script substitution branch whenever that
  // foreign script is large, hijacking this field's base/Latin bold into the
  // large-script Strong font even though the REGULAR (non-bold) text in the
  // same field correctly resolves the base/Translation family. Gating on
  // familyScript instead keeps regular vs bold in the SAME resolved script
  // (transcription's familyScript is already "latn", so it is unaffected;
  // role=source's familyScript already equals `script`, also unaffected).
  // Per-block gate scoped to THIS block's own resolved script rather than the
  // book-level `large-script` flag (book()'s global show-rule still governs
  // plain prose outside any of these 6 blocks). F3-safe: `text(font:)` sits
  // INSIDE the show rule's `it.body`, never wrapping the strong/emph element
  // from outside.
  //
  // The "else" branch finalizes via `text(weight:/style:, it.body)` rather
  // than returning `it` verbatim: Typst's show-rule chaining still offers an
  // unmodified `it` to any ENCLOSING show-strong/emph rule (verified via
  // isolated `typst query` metadata probes), so a bare `else { it }` would
  // let book()'s own book-level large-script gate re-substitute this field's
  // bold/emph whenever the BOOK itself is large-script — even though THIS
  // field's own familyScript isn't (the exact "regular vs bold land in
  // different scripts" defect, just re-introduced one level out). Explicitly
  // re-emitting via `text()` (confirmed visually identical to Typst's native
  // default strong/emph rendering) stops that leak.
  show strong: it => if _isLargeScript(familyScript) {
    context text(font: _resolveFont(script: familyScript, ext: "text", field: role, style: "strong"), weight: "regular", it.body)
  } else { text(weight: "bold", it.body) }
  show emph: it => if _isLargeScript(familyScript) {
    context text(font: _resolveFont(script: familyScript, ext: "text", field: role, style: "emphasis"), style: "normal", it.body)
  } else { text(style: "italic", it.body) }

  if role == "grammar" {
    show table: it => block(width: 100%, context text(dir: _sourceDir.get(), font: _roleFonts.get().body)[#it])
    context text(font: _resolveFont(script: "", ext: "text", field: "grammar"), body)
  } else if role == "transcription" {
    context text(font: _resolveFont(script: "latn", ext: "text", field: "transcription"), body)
  } else if role == "translation" {
    context text(font: _resolveFont(script: "", ext: "text", field: "translation"), size: _baseSize(), body)
  } else {
    // FR-1: the entire else-branch body is wrapped in context text(size:
    // _foreignSize(script), ...) below, which enlarges an Arabic/CJK/etc. ambient
    // em. A raw markdown H2 inside that scope resolves its `1.4em` level-2 size
    // against the already-enlarged em — rendering too large vs. a _blockheading
    // H2 (which is outside any _foreignSize scope). This show rule divides out the
    // factor for headings only, restoring parity with _blockheading headers.
    // - Must use the function form (it => context ...) because _foreignSizeFactor
    //   reads _sizeFactor state, which requires a context (bare `set text(size:)`
    //   form fails to compile: "can only be used when context is known").
    // - Must be scoped here (inside the else block), NOT at function top: the
    //   grammar/transcription/translation branches above NEVER apply _foreignSize,
    //   so a function-wide rule would incorrectly shrink their headings.
    show heading: it => context text(size: 1em / _foreignSizeFactor(script), it)
    context text(font: _resolveFont(script: script, ext: "text", field: "source"), size: _foreignSize(script), body)
  }
}

#let vocabulary(dir: ltr, script: "", ..items) = {
  set text(dir: dir)
  let run = ()
  for it in items.pos() {
    let k = it.at("kind", default: "data")
    if k == "header" {
      if run.len() > 0 {
        block(width: 100%, grid(
          columns: (1fr, 1fr),
          column-gutter: 1em,
          stroke: (y: 0.5pt + luma(220)),
          align: (start + top, start + top),
          inset: (x: 2pt, y: 4pt),
          ..run
        ))
        run = ()
      }
      _blockheading(it.at("level"), it.at("text"))
    } else {
      // ItemData (vocabulary has no notes, D1)
      run += (
        {
          // Phrase is the foreign/target field (SPECS §6): large-script gate
          // per this block's OWN script (no Major-2 decoupling — phrase is
          // always the block's own foreign field). The non-large branch
          // FINALIZES via `text(weight: "bold", ...)` rather than a bare
          // `strong(...)` call (SPECS §8.4 fix pass, residual item): a bare
          // strong() creates a fresh element still visible to any ENCLOSING
          // show-strong rule (book()'s own book-level large-script gate), so a
          // non-large-script phrase nested in a large-script BOOK would
          // otherwise get re-substituted with the book's Strong font — the
          // same "unfinalized element leaks to the outer gate" defect fixed
          // for textblock/dialog/parallel above, visually identical output to
          // Typst's native bold for the common (non-large-script book) case.
          if _isLargeScript(script) {
            context text(font: _resolveFont(script: script, ext: "vocabulary", field: "phrase", style: "strong"), weight: "regular", size: _foreignSize(script), it.at("phrase", default: ""))
          } else {
            text(weight: "bold", it.at("phrase", default: ""))
          }
          if it.at("grammar", default: "") != "" {
            [ ]; context text(font: _resolveFont(script: "latn", ext: "vocabulary", field: "tag"), dir: ltr, size: 0.85em, fill: gray)[#it.at("grammar")]
          }
          if it.at("transcription", default: "") != "" {
            [ ]; emph[#context text(font: _resolveFont(script: "latn", ext: "vocabulary", field: "transcription"), dir: ltr)[\[#it.at("transcription")\]]]
          }
        },
        context text(font: _resolveFont(script: "", ext: "vocabulary", field: "translation"), dir: ltr, size: _baseSize())[#it.at("translation", default: "")],
      )
    }
  }
  if run.len() > 0 {
    block(width: 100%, grid(
      columns: (1fr, 1fr),
      column-gutter: 1em,
      stroke: (y: 0.5pt + luma(220)),
      align: (start + top, start + top),
      inset: (x: 2pt, y: 4pt),
      ..run
    ))
  }
}

#let models(dir: ltr, script: "", ..items) = {
  set text(dir: dir)
  let run = ()
  for it in items.pos() {
    let k = it.at("kind", default: "data")
    if k == "header" {
      if run.len() > 0 {
        block(width: 100%, grid(
          columns: (1fr, 1fr),
          column-gutter: 1em,
          align: (start + top, start + top),
          inset: (x: 2pt, y: 4pt),
          ..run
        ))
        run = ()
      }
      _blockheading(it.at("level"), it.at("text"))
    } else if k == "note" {
      if run.len() > 0 {
        block(width: 100%, grid(
          columns: (1fr, 1fr),
          column-gutter: 1em,
          align: (start + top, start + top),
          inset: (x: 2pt, y: 4pt),
          ..run
        ))
        run = ()
      }
      _blocknote(it.at("text"))
    } else {
      let phrase = it.at("phrase", default: "")
      let transcription = it.at("transcription", default: "")
      let translation = it.at("translation", default: "")
      // Phrase is the foreign/target field (SPECS §6): large-script gate per
      // this block's OWN script (no Major-2 decoupling). Non-large branch
      // FINALIZES via `text(weight: "bold", ...)` rather than a bare
      // `strong(...)` call — see vocabulary()'s matching comment above for
      // why (SPECS §8.4 fix pass, residual item).
      let phraseContent = if _isLargeScript(script) {
        context text(font: _resolveFont(script: script, ext: "models", field: "phrase", style: "strong"), weight: "regular", size: _foreignSize(script), phrase)
      } else {
        text(weight: "bold", phrase)
      }
      if transcription == "" and translation == "" {
        run += (grid.cell(colspan: 2, phraseContent),)
      } else {
        run += (
          {
            phraseContent
            if transcription != "" and translation != "" {
              linebreak()
              emph[#context text(font: _resolveFont(script: "latn", ext: "models", field: "transcription"), dir: ltr)[\[#transcription\]]]
            }
          },
          if translation != "" {
            context text(font: _resolveFont(script: "", ext: "models", field: "translation"), dir: ltr, size: _baseSize())[#translation]
          } else if transcription != "" {
            emph[#context text(font: _resolveFont(script: "latn", ext: "models", field: "transcription"), dir: ltr)[\[#transcription\]]]
          } else { [] },
        )
      }
    }
  }
  if run.len() > 0 {
    block(width: 100%, grid(
      columns: (1fr, 1fr),
      column-gutter: 1em,
      align: (start + top, start + top),
      inset: (x: 2pt, y: 4pt),
      ..run
    ))
  }
}

// role: carries the block's as= attribute value ("source"/"translation");
// named `role`, not `as`, because `as` is a reserved Typst keyword (mirrors
// textblock's pre-existing role: convention for the same reason).
#let questions(dir: ltr, script: "", role: "source", ..items) = {
  set text(dir: dir)
  set par(first-line-indent: 0pt)
  // SPECS §5: questions accepts as=source|translation; as=translation
  // resolves question/answer via the Translation base role (Major-2: fixed
  // script, decoupled from this block's own foreign `script` param).
  let asTranslation = role == "translation"
  let familyScript = if asTranslation { "" } else { script }
  let run = ()
  for it in items.pos() {
    let k = it.at("kind", default: "data")
    if k == "header" {
      if run.len() > 0 {
        grid(columns: (auto, 1fr), column-gutter: 1em, row-gutter: 0.5em, align: (start + top, start + top), ..run)
        run = ()
      }
      _blockheading(it.at("level"), it.at("text"))
    } else if k == "note" {
      if run.len() > 0 {
        grid(columns: (auto, 1fr), column-gutter: 1em, row-gutter: 0.5em, align: (start + top, start + top), ..run)
        run = ()
      }
      _blocknote(it.at("text"))
    } else {
      let question = it.at("question", default: "")
      let answer = it.at("answer", default: "")
      if answer != "" {
        run += (
          context text(font: _resolveFont(script: familyScript, ext: "questions", field: "question", as-translation: asTranslation), size: if asTranslation { _baseSize() } else { _foreignSize(familyScript) }, question),
          context text(font: _resolveFont(script: familyScript, ext: "questions", field: "answer", as-translation: asTranslation), size: if asTranslation { _baseSize() } else { _foreignSize(familyScript) }, answer),
        )
      } else {
        if run.len() > 0 {
          grid(columns: (auto, 1fr), column-gutter: 1em, row-gutter: 0.5em, align: (start + top, start + top), ..run)
          run = ()
        }
        context text(font: _resolveFont(script: familyScript, ext: "questions", field: "question", as-translation: asTranslation), size: if asTranslation { _baseSize() } else { _foreignSize(familyScript) }, question)
        parbreak()
      }
    }
  }
  if run.len() > 0 {
    grid(columns: (auto, 1fr), column-gutter: 1em, row-gutter: 0.5em, align: (start + top, start + top), ..run)
  }
}

// role: see questions' comment above (named `role`, not `as` — reserved).
#let dialog(dir: ltr, script: "", role: "source", ..turns) = {
  set text(dir: dir)
  // SPECS §5: dialog accepts as=source|translation; as=translation
  // resolves header/content via the Translation base role (Major-2: fixed
  // script for the FAMILY chain only — the Strong gate below still reads
  // this block's own `script`, since a translated dialogue can still be
  // written in a large script).
  let asTranslation = role == "translation"
  let familyScript = if asTranslation { "" } else { script }

  // SPECS §8.4 (review-amended 2026-07-31): gate on familyScript, not the raw
  // block `script` — otherwise an as=translation dialog's header/content bold
  // (fixed-script/base family, familyScript=="") would still enter the
  // large-script substitution branch whenever the block's OWN foreign script
  // is large, picking up the large-script Strong font for what should be a
  // base/Translation-role field (the code-review defect: regular content and
  // bold landing in different scripts). The "else" branch finalizes via
  // `text(weight:/style:, ...)` rather than `it`/a fresh `strong(...)` call,
  // for the same reason documented in textblock above: an unfinalized strong
  // element is still visible to book()'s OWN book-level large-script gate,
  // which would otherwise re-substitute whenever the BOOK itself is large
  // script, regardless of this field's own (decoupled) familyScript.
  show strong: it => if _isLargeScript(familyScript) {
    context text(font: _resolveFont(script: familyScript, ext: "dialog", field: "content", style: "strong", as-translation: asTranslation), weight: "regular", it.body)
  } else { text(weight: "bold", it.body) }
  show emph: it => if _isLargeScript(familyScript) {
    context text(font: _resolveFont(script: familyScript, ext: "dialog", field: "content", style: "emphasis", as-translation: asTranslation), style: "normal", it.body)
  } else { text(style: "italic", it.body) }

  let run = ()
  for t in turns.pos() {
    let k = t.at("kind", default: "data")
    if k == "header" {
      if run.len() > 0 {
        block(width: 100%, grid(
          columns: (auto, 1fr), column-gutter: 0.8em, row-gutter: 0.5em,
          ..run
        ))
        run = ()
      }
      _blockheading(t.at("level"), t.at("text"))
    } else if k == "note" {
      if run.len() > 0 {
        block(width: 100%, grid(
          columns: (auto, 1fr), column-gutter: 0.8em, row-gutter: 0.5em,
          ..run
        ))
        run = ()
      }
      _blocknote(t.at("text"))
    } else {
      run += (
        if _isLargeScript(familyScript) {
          context text(font: _resolveFont(script: familyScript, ext: "dialog", field: "header", style: "strong", as-translation: asTranslation), weight: "regular", size: _foreignSize(familyScript), t.at("header", default: ""))
        } else {
          context text(size: if asTranslation { _baseSize() } else { 1em }, weight: "bold", t.at("header", default: ""))
        },
        context text(font: _resolveFont(script: familyScript, ext: "dialog", field: "content", as-translation: asTranslation), size: if asTranslation { _baseSize() } else { _foreignSize(familyScript) })[#t.at("content", default: [])],
      )
    }
  }
  if run.len() > 0 {
    block(width: 100%, grid(
      columns: (auto, 1fr), column-gutter: 0.8em, row-gutter: 0.5em,
      ..run
    ))
  }
}

#let parallel(source-dir: ltr, script: "", ..rows) = {
  // Column semantics (SPECS ASR-1, deliberate reversal of the shipped
  // SR-1/SR-2 rule): the PRIMARY column (element 1) now carries the SOURCE —
  // marker-driven direction (`source-dir`), font, and large-script size
  // (_foreignSize, ASR-7). The SECONDARY column (element 2) carries the
  // TRANSLATION — book language, no marker override; it inherits the book's
  // ambient direction (no `dir:` override here, ASR-4).
  //
  // Per-column gate placement:
  // - SOURCE: the large-script strong/emph gate (keyed `field:"source"` +
  //   the marker `script`) is installed inside an INNER scope so it cannot
  //   capture bold/emph in the stacked transcription below (ASR-8). The
  //   "else" branch finalizes via `text(weight:/style:, ...)` (not bare `it`)
  //   so a non-large source nested in a large-script book cannot leak to
  //   book()'s outer gate (same reasoning as textblock/dialog/vocabulary).
  // - TRANSCRIPTION: stacked below the source inside the primary cell,
  //   OUTSIDE the source's inner scope so its bold/emph falls through to
  //   book()'s book-level gate. Pinned script:"latn"/dir:ltr (matches
  //   vocabulary/models transcription exactly, ASR-6). Present only when the
  //   row dict carries the "transcription" key — the Go renderer omits the
  //   key entirely when absent (§7.1), so dict-key membership is the correct
  //   check here (reviewer-flagged idiom, SPECS §7.2).
  // - TRANSLATION: no scoped gate — its bold/emph falls through to book()'s
  //   book-level large-script gate, exactly as the old "main" element did
  //   ("parallel main -> book", deliberately absent, not a separate
  //   mechanism). No `dir:` override — inherits book ambient direction
  //   (ASR-1/ASR-4 reversal: the old secondary carried secondary-dir; the
  //   new translation inherits book direction).
  block(width: 100%, grid(
    columns: (1fr, 1fr), column-gutter: 1.2em, row-gutter: 0.5em,
    stroke: (x: 0.5pt + luma(230)),
    align: (start + top, start + top),
    ..rows.pos().map(r => (
      {
        // SOURCE — inner scope so the gate below cannot bleed into the
        // transcription stacked after this scope closes (ASR-8).
        {
          show strong: it => if _isLargeScript(script) {
            context text(font: _resolveFont(script: script, ext: "parallel", field: "source", style: "strong"), weight: "regular", it.body)
          } else { text(weight: "bold", it.body) }
          show emph: it => if _isLargeScript(script) {
            context text(font: _resolveFont(script: script, ext: "parallel", field: "source", style: "emphasis"), style: "normal", it.body)
          } else { text(style: "italic", it.body) }
          context text(dir: source-dir, font: _resolveFont(script: script, ext: "parallel", field: "source"), size: _foreignSize(script))[#r.at("source", default: [])]
        }
        // TRANSCRIPTION (pinned latn/ltr, matches vocabulary/models, ASR-6)
        // — outside the source scope so its bold/emph falls through to
        // book()'s book-level gate (ASR-8).
        if "transcription" in r {
          linebreak()
          context text(dir: ltr, font: _resolveFont(script: "latn", ext: "parallel", field: "transcription"))[#r.at("transcription")]
        }
      },
      // TRANSLATION — no scoped gate; bold/emph falls through to book()'s
      // book-level large-script gate (deliberately absent, not a separate
      // mechanism). No dir: override — inherits book ambient direction.
      context text(font: _resolveFont(script: "", ext: "parallel", field: "translation"), size: _baseSize())[#r.at("translation", default: [])],
    )).flatten()
  ))
}

#let book(
  title: none,
  author: none,
  description: none,
  lang: "en",
  dir: ltr,
  cover: none,
  paper: "a5",
  size: 12pt,
  size-large: 16pt,
  large-script: false,
  margin: (inside: 1.8cm, outside: 1.4cm, top: 1.7cm, bottom: 2cm),
  font: _baseFont,
  font-body: (),
  font-header: (),
  font-transcription: (),
  font-translation: (),
  font-strong: (),
  font-emph: (),
  font-notes: (),
  font-slots: (:),
  book-script: "",
  contents-title: [Contents],
  body,
) = {
  let headerFont = font-header + font
  let strongFont = font-strong + font
  let emphFont = font-emph + font
  _roleFonts.update((
    body: font-body + font,
    header: headerFont,
    transcription: font-transcription + font,
    translation: font-translation + font,
    strong: strongFont,
    emph: emphFont,
    notes: if font-notes == () { emphFont } else { font-notes + font },
  ))
  _sourceDir.update(dir)
  _fontSlots.update(font-slots)
  _bookScript.update(book-script)
  // 1.0 keeps a whole-book large-script book unchanged (its base is already
  // size-large); otherwise foreign runs scale by size-large/size (SPECS FR-2).
  _sizeFactor.update(if large-script { 1.0 } else { size-large / size })
  // Inverse ratio: divides the enlarged ambient back to base `size` for
  // translation-role fields in a large-script book. 1.0 when not large-script
  // (ambient is already `size`, no adjustment needed).
  _baseSizeFactor.update(if large-script { size / size-large } else { 1.0 })

  set document(title: title, author: if author == none or author == "" { () } else { author })
  set text(
    lang: lang,
    dir: dir,
    size: if large-script { size-large } else { size },
    font: font-body + font,
    hyphenate: true,
  )
  set page(paper: paper, margin: margin, numbering: "1")
  set par(justify: true, leading: 0.7em, spacing: 0.7em, first-line-indent: (amount: 1.2em, all: false))
  show heading: set par(justify: false, first-line-indent: 0pt)
  set terms(separator: [: ], tight: true, hanging-indent: 1em)
  set table(
    stroke: (x: none, y: 0.7pt + luma(180)),
    inset: (x: 8pt, y: 5pt),
    align: (_, y) => if y == 0 { center } else { left },
    fill: (_, y) => if y == 0 { luma(235) } else { none },
  )
  show heading: set block(above: 1.5em, below: 0.75em)
  show heading: set text(font: headerFont)
  show heading.where(level: 1): set text(size: 1.7em)
  show heading.where(level: 2): set text(size: 1.4em)
  show heading.where(level: 3): set text(size: 1.2em)
  show heading.where(level: 4): set text(size: 1.1em)
  show heading.where(level: 5): set text(size: 1.05em)
  show heading.where(level: 6): set text(size: 1em)
  // Book-level gate: governs plain prose OUTSIDE any of the 6 custom
  // blocks above (each of which now installs its OWN per-block gate,
  // SPECS §8.4/INC2.5, that supersedes this one within its own scope).
  show strong: it => if large-script { text(font: strongFont, weight: "regular", it.body) } else { it }
  show emph: it => if large-script { text(font: emphFont, style: "normal", it.body) } else { it }
  show heading: it => if large-script { set text(weight: "regular"); it } else { it }
  show quote.where(block: true): it => block(inset: (left: 1em, y: 0.3em), stroke: (left: 2pt + luma(180)))[#emph(it.body)]

  set page(numbering: none)

  if cover != none {
    page(margin: 0pt, numbering: none, {
      if type(cover) == str {
        image(cover, width: 100%, height: 100%, fit: "cover")
      } else { cover }
    })
  }

  {
    set par(justify: false)
    set text(hyphenate: false)
    align(center + horizon, {
      if large-script {
        text(size: 2.4em, font: strongFont, title)
      } else {
        text(size: 2.4em, weight: "bold", title)
      }
      if author != none and author != "" { v(1.2em); text(size: 1.3em, author) }
      if description != none and description != "" {
        v(1.6em)
        if large-script {
          text(font: emphFont, fill: luma(90%), description)
        } else {
          text(style: "italic", fill: luma(90%), description)
        }
      }
    })
  }
  pagebreak()

  set page(numbering: "i")
  counter(page).update(1)
  show outline.entry.where(level: 1): strong
  outline(title: contents-title, indent: auto)
  pagebreak()

  set page(numbering: "1")
  counter(page).update(1)

  body
}
