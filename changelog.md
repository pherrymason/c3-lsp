# C3LSP Release Notes

## Unreleased

- Hover/import resolution: fixed cold-hover reliability for packaged and nested workspaces (notably `bindgen.c3l`) so imported identifiers like `BGOptions`, qualified imported calls like `bgstr::is_between`, and short qualified module paths like `wter::...`/`ttor::...`/`check::...` resolve on the first hover instead of returning synthetic/null results or requiring warm-up hovers.
- Hover correctness: module-token positions (`foo` in `foo::bar`) are now strict module lookups only, with normalized document-ID-to-URI fallback/loading paths and tighter import-root materialization, preventing incorrect fallback to unrelated symbols/macros such as resolving `check::...` to stdlib `@check`.
- Import preload parsing: fixed module declaration extraction to ignore prose in doc comments (e.g. `module wrapping`) and only match real `module ...;` declarations, restoring discovery for modules like `bgimpl::ttor` declared in `translators.c3`.
- Library/indexing support: added support for both `.c3l` directory packages and `.c3l` ZIP archives in workspace/dependency indexing, including archive-backed virtual `*.c3i` documents, `.c3l` file watching, and real-workspace validation against `sqlite3.c3l` and `bindgen.c3l`.
- Workspace/indexing UX: added richer IDE work-done progress reporting for stdlib loading, synchronous workspace indexing, and first-hover-triggered indexing, with clearer phases for root resolution, scanning, source loading, document indexing, and snapshot finalization.
- Grammar compatibility: aligned internal symbol model names with C3 0.7.11 grammar by renaming `Def` -> `Alias`, `Distinct` -> `TypeDef`, and `Fault` -> `FaultDef`, and updated parser/search/hover/test callers accordingly.
- Parser/attributes: moved shared parser node helpers into a common helper file and applied declaration attribute parsing consistently across aliases, typedefs, faultdefs, structs, enums, bitstructs, and interfaces.
- Tech debt/cleanup: removed deprecated document-reader helpers, dropped the test-only `DocumentStore` normalization fallback, cleaned stale TODO/dead-comment leftovers from parser and search paths, and expanded hover/import regression coverage for cold-index behavior, declaration-vs-filename module discovery, and real-workspace packaged-library resolution.

- Stability/requests: added deterministic request guardrails across hot paths (timeouts, inflight backpressure, watchdog correlation with `request_id`, slow-request summaries, and optional goroutine dumps) to prevent hangs and make latency behavior predictable under load.
- Stability/search: added bounded search traversal controls (deadline/depth caps) and guarded `documentHighlight` execution to stop deep recursive resolution from monopolizing the server.
- Stability/config reload: workspace config changes are now applied live when `c3lsp.json`/`project.json` changes are detected via `workspace/didChangeWatchedFiles` or `textDocument/didSave`, including reconfiguration of logger output when `log-path` changes.
- Hover/typing: improved unwrap-binding type inference so untyped `try`/`catch` bindings (e.g. conversion-call patterns like `to_integer(uint, ...)`) now carry inferred types, and hover displays typed variables (`uint n` instead of bare `n`).
- Hover/navigation: fixed separator-cursor resolution for qualified symbols (`module::symbol`) by preferring right-side retry probes, avoiding incorrect hover fallbacks such as resolving `log::error` to unrelated `std::math::log`.
- Indexing/dependencies: workspace indexing now includes `project.json` `dependency-search-paths` (resolved and deduplicated), so hover/definition can resolve symbols from external dependency roots (e.g. `sqlite3` in `../vendor/libraries`) without opening a broader workspace.
- Hover/constants: constant hover signatures now include the `const` keyword and initializer value (`const <type> NAME = <value>`) for all indexed constant symbols.
- Hover/enum values: enum and `constdef` members now display explicit assigned values using const-style formatting with enum context (`enum EnumName.MEMBER = VALUE`), and associated enum members render typed payload details (`enum EnumName.MEMBER {Type name: value}`).
- Hover/unwrap inference: untyped `try`/`catch` binding variables now infer hover types from resolved function/method return types (including optional unwrap for `try`), and non-local inferred types are rendered fully-qualified (e.g. `sqlite3::SqliteStmt s`).
- Hover/faultability: optional-return function hover now lists `@faults` from `@return`/`@return?` contracts, with an `@faults (inferred)` fallback based on `return FAULT~` and one-level `call(...)!` propagation when explicit fault contracts are absent; fault names are qualified where possible (including `@builtin::...`).
- Hover/fault symbols: bare builtin fault constants in expressions (e.g. `TYPE_MISMATCH~`) now resolve reliably through builtin fallback lookup, preventing `hover = null` in optional-return code paths.
- Hover/parser: fixed method-receiver type inference for by-reference receivers not named `self` (first receiver param only), so member/method hover resolution works for patterns like `fn void Router.free(&router)` and `router.routes.free()`.
- Hover/parser: indexed `foreach` iterator bindings as local variables and infer iterator types from iterable expressions (including receiver member collections like `router.routes`), so hover/symbol resolution now works on iterator names and member access (e.g. `route` and `route.path` in `foreach (route : router.routes)`).
- Hover/parser: fixed `foreach (index, element : iterable)` typing so the first binding resolves as `usz` index and the second as element type, preventing hover from mis-typing index variables (e.g. `WaitEntry i`).
- Hover/parser: aligned unwrap-binding semantics for `catch` by typing untyped catch-bindings as `fault` (instead of inferred success value), so hover now reflects C3 optional-fault behavior in patterns like `if (catch reason = optional_value)`.
- Hover/unwrap inference: `try` bindings now infer from optional-valued variables/member expressions on the RHS (not only direct call expressions), so hover resolves unwrapped types in patterns like `if (try accepted = accepted_result)`.
- Hover robustness: when direct lookup misses, hover now retries symbol resolution at the detected token boundaries (start/end of identifier) before punctuation-neighbor fallback, improving stability for cursor positions inside method names in casted/member-call expressions.
- Search/type resolution: hardened one-level type lookup to avoid prefix-collision picks and to recover short-module type qualifiers (e.g. `tcp::TcpSocket` -> `std::net::tcp::TcpSocket`) during member-chain resolution, improving hover/definition on calls like `client.socket.close()`.
- Stdlib/cache: bumped cache format version to force stdlib reindex with the latest method-owner parsing/resolution fixes and avoid stale cached method symbols (e.g. owner-qualified `Socket.close`).
- Hover/parser: fixed method owner extraction for optional-return method signatures (e.g. `fn void? Socket.close(...)`) so methods are indexed with owner-qualified names and hover resolves calls like `client.socket.close()`; stdlib cache format bumped to force rebuild with corrected symbols.
- Stdlib/cache UX: introduced cross-process stdlib cache lockfiles (`stdlib_<version>.json.lock`) with stale-lock recovery plus atomic temp-file cache writes to prevent concurrent cache corruption/races across multiple LSP processes.
- Tests/builtin coverage: added strict accessibility matrix tests for all discovered `@builtin` std symbols, asserting each symbol resolves either after importing its defining module or via global accessibility fallback.

- Performance/rename: removed redundant comment/string filtering in references-backed rename edit aggregation (the references engine already filters these), eliminating repeated full-source rescans per reference and significantly improving large-file rename latency.
- Performance/rename: removed redundant deduplication in references-backed rename edit aggregation and switched reference-location dedupe keys to a typed struct (reducing allocation pressure on high-match workloads).
- Performance/rename: added struct-member semantic-rename fast path to short-circuit expensive fallback declaration resolution for validated owner-typed member accesses; references microbenchmark also improved.
- Performance/rename: optimized references-backed rename by speeding up `FindReferencesInWorkspace` (token scan, comment/string filtering, incremental position mapping, and per-doc access memoization), reducing large-file rename latency by roughly 60-75% in local benchmarks.
- Performance/rename: optimized rename hot paths with precomputed comment/string spans, incremental byte->LSP position mapping, and boundary-aware token scanning (removing per-call regex compile for candidate matching); large-file rename benchmarks improved (~4-6%).
- Verification: re-ran targeted lifecycle/diagnostics integration tests and completed a VS Code client sanity build (`vscode:prepublish`) to validate end-to-end compatibility expectations.
- Stability/completion: added nil-safe unit-module guards in completion context/field resolution paths to prevent request-path panics when parsed-module snapshots are temporarily unavailable.
- Search architecture: added shared one-level symbol resolver (`search/resolve_shared.go`) and wired both search v1 and search v2 paths to it to reduce resolver drift.
- Maintainability/completion: decomposed completion handling into explicit context + render pipeline stages (`completion_pipeline.go`) and simplified `TextDocumentCompletion` orchestration.
- Maintainability/rename: extracted rename conflict detection/validation logic into `rename_conflicts.go` to reduce `TextDocumentRename.go` complexity.
- Performance/tests: added completion benchmark smoke coverage (`BenchmarkTextDocumentCompletion_InScope`) to complement existing rename/references performance guards.
- Performance/indexing: avoided triggering full async workspace scans from request hot paths for non-buildable roots (no `project.json`), reducing unnecessary scan churn.
- Diagnostics/cancellation: made compiler diagnostics execution context-aware and cancel previous in-flight diagnostics command when a newer run starts.
- Diagnostics: made diagnostic-clear publication synchronous (instead of goroutine fire-and-forget) to reduce stale/out-of-order publish risk under rapid edit bursts.
- Text scanning: moved document symbol-boundary scanning to rune-safe UTF-8 traversal (avoiding byte-index rune coercion) and added deterministic incremental-edit application tests.
- Protocol errors: centralized lifecycle/request guard error values and assertions for consistent server-side error semantics across handlers.
- Tests/positioning: added table-driven coverage for UTF-16 position mapping across unicode, CRLF line boundaries, and tab characters.
- Position/range correctness: improved UTF-16 index clamping for CRLF lines and fixed multi-line range boundary checks; added regression tests for surrogate pairs, CRLF, and boundary predicates.
- Concurrency/tests: added coverage for configuration changes during active indexing to ensure in-flight async indexing is cancelled safely before reindex.
- State/lifecycle: kept `DocumentStore` in sync during `ProjectState` delete/rename flows so document lifecycle mutations now update document storage together with symbol/index state.
- Tooling/tests: added `make test-race-server` to run race detection across core LSP packages (`internal/lsp/server`, `project_state`, `search`, `search_v2`).
- State/maintainability: documented `ProjectState` lock ownership and lock-ordering rules (`stateMu` vs `parseMu`) to reduce future concurrency regressions.
- Concurrency/index-state: centralized root/index map ownership under a single indexing-state lock path, removing split lazy-initialization in unrelated code paths.
- Text sync/lifecycle: added document version tracking and stale `didChange` rejection so out-of-order updates are ignored; also hardened `didOpen`/`didChange`/`didSave`/`didClose` with lifecycle and nil-parameter guards.
- Lifecycle/protocol: standardized guarded request execution (`runGuardedRequest`) across remaining request handlers (including workspace file-operation requests and completion-item resolve), and added lifecycle-aware notification guards for workspace events.
- Lifecycle/protocol: added explicit `exit` handler wiring and shutdown-order validation (`exit` before `shutdown` now returns a protocol error), plus server-side lifecycle tracking for shutdown/exit transitions.
- Lifecycle/protocol: added explicit initialize/shutdown lifecycle gating so requests are rejected before `initialized` and after `shutdown`, and notifications (`didOpen`/`didChange`/`didSave`/`didClose`) are ignored outside active session state.
- Stability/core: fixed `symbols.Node.Insert` child-linking, corrected `--diagnostics-delay` CLI units to milliseconds, hardened `ProjectState` rename/close safety (nil/locking), tightened parent-resolution cache keys with cursor position, and aligned search v1/v2 distinct-context constants to prevent cross-engine semantic drift.
- Stability/Zed: hardened request handling with per-request panic recovery for hover/definition/declaration/typeDefinition/implementation/completion/signatureHelp/rename, so single bad requests no longer kill the LSP process.
- Stability/Hover: fixed multiple nil-deref paths when hovering symbols (including typed-nil search results and unresolved distinct base types like `Thread`), returning safe empty hover/error responses instead of crashing.
- Stability/position parsing: replaced several out-of-bounds panics in document/source parsing and cursor-position math with safe bounds checks/clamping and fallback returns.
- Stability/path parsing: `NormalizePath` now falls back to canonical raw paths when URI parsing fails, instead of panicking.
- Grammar compatibility: added support for the new C3 `constdef` keyword (keywords/completion/tests) while keeping parser compatibility by normalizing `constdef` during CST parsing for the current vendored tree-sitter runtime.

- Workspace/navigation: improved aggregate-folder support (e.g. opening `/Users/.../c3`) by resolving the nearest project root per active document, deferring heavy root-wide indexing on non-project roots, and indexing subprojects on demand for go-to-definition/hover.
- Diagnostics/configuration: diagnostics now run from the active file's nearest project root (with per-project config reload), instead of always using the opened workspace root.
- Navigation compatibility: added `textDocument/typeDefinition` support and more robust document-loading guards for editors that send different navigation requests or delayed open/index events.
- Zed compatibility: relaxed C3 document detection in `didOpen` (case-insensitive language id + C3 extension fallback) to avoid missed indexing when clients report `C3`/nonstandard ids.
- Navigation: go-to-definition/declaration now retries symbol resolution one or two characters to the left when the cursor lands on trailing call punctuation (e.g. `name|(`), fixing missed jumps such as `stress::run_fiber_backend_repro(...)`.
- Completion: fixed `Ctrl+Space` on empty lines so in-scope suggestions are shown instead of being filtered by previous-line symbols.
- Completion: improved empty-invoke ordering with scope-aware ranking (`SortText`) so local symbols rank above modules and language keywords, with `$...` keywords ranked last.
- Completion UI metadata: enriched completion rows with `labelDetails.description` (kind hints) and signature markdown for callable/type items to improve editor-side highlighting.
- Completion UI metadata: aligned `labelDetails` mapping with Zed Rust expectations (`description` carries signature detail, `detail` carries kind hint) for better list rendering.
- Completion: imported module path suggestions now hide non-public symbols (`@private`, `@local`) outside valid visibility contexts.
- Completion: `@local` symbols are scoped by module declaration section; locals from `module X` part 1 no longer leak into `module X` part 2.
- Completion accept: improved callable and struct insertion snippets, including context-aware declaration/value struct snippets and robust replacement ranges.
- Completion accept: method completion insertion now strips type qualifiers (`Type.method`) so instance calls insert correctly as `obj.method()`.
- Completion: improved chain completion context detection for `Ctrl+Space` around dot access (`obj.|`, `obj|.`, and next-line after `obj.`), including safer symbol-at-cursor fallback.
- Completion: improved stdlib chain completion on incomplete lines by adding a type-inference fallback for unresolved `obj.` contexts (e.g. `List{int} l; l.` / `HashMap ... v; v.`).
- Navigation: go-to-definition/declaration now resolves `@`-prefixed macro symbols when clicked from usages that omit/strip `@` in token extraction.
- Navigation: improved short module-path resolution for qualified calls (e.g. `types::...`, `runtime::...`) by adding fallback matching to indexed modules with `std::core::*` precedence when ambiguous.
- Hover: generic type rendering now includes concrete type arguments in signatures/details (e.g. `HashMap{String, Feature}`, `List{int}`) for variables, members, defs, distincts, function signatures, and type-identifier hover on generic instantiations (e.g. hovering `List` in `List{int}`).
- Hover/Completion docs: module generics now surface inferred per-parameter constraints from module `@require` contracts (e.g. `Key` constrained in `std::collections::map<Key, Value>`), with unconstrained parameters explicitly shown.
- Hover: module symbols now include declared generic parameters in the signature line (e.g. `std::collections::map <Key, Value>`).
- Hover/Completion docs: generic module constraints now render in the existing contract style (`@require ...`) and avoid duplicate display on direct module hover.
- Completion accept: struct-construction snippet expansion is now suppressed in generic type-argument contexts (e.g. `HashMap{Tile, int}`), preventing invalid replacements like `Tile t = {...}` where only a type is expected.
- Completion list: root symbol completion no longer suggests type methods (e.g. `Tile.print_tile`) outside member-access contexts; methods are now suggested only for valid receiver chains like `tile.`.
- Navigation: go-to-definition/declaration now ignores literal contexts (e.g. `%s` inside string literals), preventing incorrect jumps to unrelated one-letter symbols like `alias s`.
- Hover: hover lookup now ignores literal contexts (e.g. `%s` inside string literals), preventing unrelated symbol docs from appearing for format-specifier characters.
- Navigation: added `textDocument/implementation` support for interfaces and interface methods (find implementors and method implementations across workspace/stdlib).
- Navigation: added initial `textDocument/references` support with include-declaration filtering and semantic parity with current rename resolution/edit matching.
- Navigation: improved references matching for enum/fault constants with qualified owner and module-path fallbacks (e.g. `Sig.SETMASK`, `blem::net::TIMEOUT`), and enabled references-backed rename for those symbol kinds.
- Navigation: added `textDocument/documentHighlight` using semantic references, with same-document filtering and declaration/write highlighting.
- Navigation: added `textDocument/documentSymbol` with hierarchical module/type/member/function outlines.
- Navigation: added `workspace/symbol` search over indexed workspace symbols with stable ordering and query filtering.
- Navigation: added `textDocument/foldingRange` for symbol regions and multi-line block/doc comments.
- Navigation: added `textDocument/selectionRange` with nested symbol-range chains for cursor expansion.
- Formatting: added `textDocument/rangeFormatting` and `textDocument/onTypeFormatting` support on top of `c3fmt`, with on-type triggers for `}` and `;`.
- Performance/completion: added phase-level `PERF_TRACE` telemetry for completion (`build_context`, `search_build_list`, `render_items`, total) plus per-request counters (`suggestions_in/out`, `struct_field_lookups`, `dedup_count`), and memoized struct-field resolution per request to avoid repeated workspace scans during snippet enrichment.
- Performance/completion: reduced render-path overhead by reusing per-request struct mode/trailing-token decisions and introduced a bounded completion response cache keyed by document version, state revision, cursor/trigger context, and symbol-at-cursor, with regression tests for repeated-request hits and version-change invalidation.
- Stability/completion: request cancellation is now threaded through the completion request pipeline (context build, pre-search, and render loop checkpoints) so stale typing-triggered requests can exit early without consuming extra CPU.
- Performance/tests: added workspace-style warm completion benchmark coverage (`BenchmarkWorkspaceCompletionWarm`) alongside the existing in-scope completion microbenchmark to track real-workspace completion latency/alloc trends.
- Performance/save diagnostics: `textDocument/didSave` now schedules quick diagnostics immediately and defers full diagnostics to a save-idle debounce window (configurable via `Diagnostics.save-full-idle-ms`, default 10000ms), reducing save-path latency spikes during rapid save bursts.
- Performance/save diagnostics: repeated saves of the same document version now skip redundant diagnostics scheduling, and save-idle full diagnostics now respect a configurable minimum interval (`Diagnostics.full-min-interval-ms`, default 30000ms) to reduce full-build churn.
- Performance/benchmarks: added save-path burst benchmarks (`BenchmarkTextDocumentDidSaveBurst_UnchangedVersion`, `BenchmarkTextDocumentDidSaveBurst_VersionBump`) to track didSave fast-path and version-bump save overhead trends.
- Performance/diagnostics scheduling: diagnostics execution is now queued per project root with latest-pending coalescing, reducing redundant queued runs during save/edit bursts while preserving the most recent diagnostics request.
- Performance/benchmarks: added `BenchmarkTextDocumentDidSaveBurst_DiagnosticsMocked` to quantify didSave burst behavior with diagnostics enabled and mocked command latency; benchmark now reports diagnostics command run ratios per save burst.
- Performance/diagnostics logging: removed unconditional compiler output logging in diagnostics execution path to reduce save-path overhead and log noise under high-frequency saves.
- Performance/root resolution: added server-side project-root resolution cache for URI lookups, with invalidation on workspace/config/file-change events, reducing repeated filesystem traversal in hot save/format/diagnostics paths.
- Telemetry/root cache: diagnostics and didSave `PERF_TRACE` logs now include cumulative `root_cache_hits` and `root_cache_misses` counters to make cache hit-rate verification easier in real sessions.
- Performance/didSave fast path: reduced save-path overhead by avoiding unconditional perf tracing setup when `PERF_TRACE` is disabled, reusing a shared noop notify callback, and using normalized-path document lookups to skip duplicate URI normalization on save checks.
- Performance/didSave normalization: added a bounded URI->normalized document ID cache in the server hot path, significantly reducing unchanged-save and diagnostics-mocked save-burst overhead/allocations.
- Performance/text sync: `didChange` and `didClose` now reuse cached normalized document IDs and call normalized-ID state APIs directly; `didOpen`/`didChange` also reuse shared noop notification handling for nil-context fast paths.
- Performance/benchmarks: added `BenchmarkTextDocumentDidChange_EmptyChange` to track no-op change notification overhead and keep text-sync fast-path improvements measurable.
- Performance/didChange fast path: empty `didChange` updates now update version in-place under lock without re-setting/re-normalizing document entries, and skip diagnostics scheduling when there are no content changes; this further reduced no-op change overhead and allocations.
- Performance/diagnostics queue: added enqueue dedupe fingerprints (`root|mode|trigger|stateRevision`) so equivalent diagnostics requests are dropped while preserving latest-wins behavior for non-equivalent pending work per root.
- Diagnostics config hardening: `Diagnostics.save-full-idle-ms` and `Diagnostics.full-min-interval-ms` are now clamped to safe ranges (`500..120000` and `1000..600000` respectively) with warning logs when values are out of bounds.
- Telemetry/root cache: diagnostics and didSave perf logs now include per-operation root-cache deltas and cumulative hit-rate (`root_cache_hits_delta`, `root_cache_misses_delta`, `root_cache_hit_rate`) for easier session-level validation.
- Diagnostics worker lifecycle: per-root diagnostics workers now retire after idle timeout, preventing worker-map growth from inactive roots while keeping queue behavior race-safe.
- Tests/diagnostics stability: added regression coverage for save-idle vs full-min-interval interaction and root-cache telemetry formatting fields to keep new scheduling/observability behavior stable.
- Performance/e2e harness: added `BenchmarkTextDocumentDidChangeSaveBurst_DiagnosticsMocked` to simulate mixed `didChange`+`didSave` bursts with mocked diagnostics latency, reporting burst p50/p95 (`burst_p50_us`, `burst_p95_us`), `diag_runs/save`, and `root_cache_hit_rate`.
- Performance/guardrails: added `bin/perf_gate.sh` to run key save/change benchmarks and fail when thresholds regress (`didChange` empty-change ns/op, `didSave` version-bump ns/op, burst p95, `diag_runs/save`, and root-cache hit-rate).
- Performance/trend tracking: added git-controlled metrics snapshot `docs/perf-metrics.json` plus `bin/perf_metrics_update.sh` to refresh benchmark numbers; perf gate now prints deltas against the tracked baseline for iteration-over-iteration visibility.
- Performance/LSP-spec tracking: added `bin/lsp_perf_metrics_update.sh` and git-tracked `docs/lsp-perf-metrics.json` to keep per-method performance status aligned with `LSP_SPEC.md` implemented entries (measured vs tracked/outbound/framework) and preserve metrics history in git.
- Workspace notifications: implemented `workspace/didChangeWorkspaceFolders` root lifecycle handling (add/remove tracking, removed-root indexing cancelation, added-root reindex scheduling) and expanded `workspace/didChangeWatchedFiles` to refresh changed source documents and trigger scoped reindexing for external C3 file changes.
- Save lifecycle hooks: implemented `textDocument/willSave` and `textDocument/willSaveWaitUntil` with deterministic default behavior (no edits unless explicitly enabled via `Formatting.will-save-wait-until`), and added config/runtime parsing plus tests for default and enabled paths.
- Workspace commands: implemented `workspace/executeCommand` (`c3lsp.reindexWorkspace`, `c3lsp.reloadConfiguration`, `c3lsp.clearDiagnosticsCache`) plus internal `workspace/applyEdit` request helper wired into the diagnostics-cache command flow.
- Window UX helpers: added wrappers for `window/showMessageRequest` and `window/showDocument` and wired an actionable prompt flow (`reloadConfiguration` command can offer opening `project.json`).
- Navigation/editing: added `textDocument/linkedEditingRange` support for deterministic identifier-linked ranges on symbol/module targets within the active document.
- Workspace symbols: added compatibility handling for `workspaceSymbol/resolve` via protocol extension dispatch (deterministic symbol pass-through for clients that request symbol resolution).
- Progress protocol: added `window/workDoneProgress/create` request helper and `window/workDoneProgress/cancel` handling with deterministic token cancellation tracking.
- Symbol identity: added `textDocument/moniker` support with deterministic moniker identifiers and stable kind/uniqueness mapping for module and symbol targets.
- Call hierarchy: added `textDocument/prepareCallHierarchy`, `callHierarchy/incomingCalls`, and `callHierarchy/outgoingCalls` support for function symbols with deterministic reference-based caller/callee mapping.
- Code actions: added deterministic baseline handlers for `textDocument/codeAction` (empty stable result) and `codeAction/resolve` (pass-through), including capability wiring.
- Performance/text sync: empty `textDocument/didChange` updates now skip unnecessary reparse/reindex work while still advancing document version, reducing no-op edit overhead.
- Formatting/config: formatting requests now resolve and apply nearest project-root configuration before invoking formatter, improving mixed-workspace behavior for per-project `Formatting` settings.
- Navigation: added `textDocument/documentLink` and `documentLink/resolve` for module-path links across import/module declarations and qualified module usages.
- Workspace lifecycle: added `workspace/willCreateFiles`, `workspace/didCreateFiles`, `workspace/willRenameFiles`, and `workspace/willDeleteFiles` handlers with C3 file-operation registration and create-file indexing refresh.
- Workspace initialization: added `workspace/workspaceFolders` request fallback when root URI/path is not provided, and advertised workspace-folders capability with change notifications.
- Telemetry: added `telemetry/event` notifications for initialize lifecycle milestones.
- Rename/references: added struct-member access fallback matching in references search and enabled references-backed struct-member rename for non-colliding cases, while preserving hardened fallback when module functions share the same member name.
- Rename compatibility: added optional `WorkspaceEdit.documentChanges` output when client capabilities advertise support, while keeping `changes` populated for backward compatibility.
- Configuration: added runtime settings refresh via `workspace/didChangeConfiguration` + `workspace/configuration` (supports `C3`/`c3` and `Diagnostics`/`diagnostics` sections).
- Stdlib indexing: improved cache robustness with cache-format versioning plus module rehydration/merge to preserve symbol relationships after reload.
- Server capabilities: initialize now advertises implementation support and emits workspace/diagnostics status messages to the client window/log channels.
- Rename: added module rename support via LSP `textDocument/prepareRename` and `textDocument/rename`, updating module declarations/imports/qualified usages.
- Rename: expanded semantic rename beyond module-only support to handle alias/`constdef` symbols and their qualified member usages (e.g. `Sig.SETMASK`, `Sig.UNBLOCK`) from both declaration and usage sites.
- Rename: improved member propagation for struct access chains (e.g. `fiber.entry`, `running.entry()`) with scope-aware fallbacks, including shadowed-identifier safety so unrelated locals/params are not renamed.
- Rename safety/compatibility: fixed `textDocument/rename` response shape for strict clients (stable empty `WorkspaceEdit` object + URI-safe change keys), and prevented non-module renames from rewriting module-path tokens (`allocator::...`) when names collide with struct members.
- Rename: fixed fault constant rename propagation for qualified module-path usages (e.g. `blem::net::TIMEOUT`) so renaming a `faultdef` member updates both declaration and qualified references, including cross-file references.
- Rename: fixed qualified module-path fallback matching for fault constants so cross-file references like `blem::net::TIMEOUT` are renamed correctly even when the usage file does not import `blem::net`.
- Rename: module rename now targets the module segment under cursor (e.g. `net` in `blem::net`) with segment placeholder/range in prepare-rename, while still supporting full-path module replacement when a full module path is provided as the new name.
- Rename safety: added pre-rename conflict validation to reject colliding names in the same scope/owner (functions, variables, struct members, enum members, and fault constants) with deterministic `rename conflict` errors.
- Rename: fixed struct-member rename targeting when cursor is on nearby type tokens in dense declarations (including multi-variant `struct Fiber` blocks); rename probe fallback now stays on the current line to avoid incorrectly selecting adjacent members.
- Rename: fixed `struct` member declaration renames when member names collide with same-module function names (e.g. `Fiber.done` vs `fn done()`), ensuring declaration + member usages update together without server crashes/disconnects.
- Rename performance: added request-scoped caching for declaration lookups and struct-owner resolution, plus member-specific candidate pruning (`.` access prefilter) to reduce rename latency on large/ambiguous files.
- Rename observability: when `PERF_TRACE` is enabled, rename now logs phase timings and cache hit ratios for easier hotspot diagnosis.
- Rename safety: rename now ignores comment/string-literal contexts for target detection and edit application, preventing accidental textual rewrites of non-code content.
- Rename UX/safety: invalid rename requests now return kind-specific errors (e.g. `invalid function name`, `invalid struct member name`), and cursor-on-comment rename requests resolve to safe no-op edits.
- Rename scalability: added large/high-match benchmarks for function and struct-member rename workloads to catch regressions on dense files.
- Rename indexing awareness: when rename runs before workspace indexing completes, the server emits a one-time per-root warning that results may be partial.
- Rename diagnostics: added `RENAME_DEBUG` feature-flagged logging for rename path decisions (target detection and references-backed/fallback selection).
- Docs: added rename troubleshooting guide covering no-op/validation/conflict outcomes, partial-index warnings, and debug flag usage.
- Rename coverage: local variables introduced by `try`/`catch` unwrap bindings (e.g. `if (try n = ...)`, `if (catch err = ...)`) are now indexed and renameable with scope-safe behavior.
- Rename docs: parameter rename now updates matching function doc contracts in adjacent doc comments (`@param <name>`, `@param [in] <name>`, and identifier occurrences in `@require` expressions).
- Rename UX: `textDocument/prepareRename` now consistently returns non-empty placeholders for function/module targets (including module-segment rename and callsite punctuation positions like `name|(`).
- Rename behavior: unsupported/ambiguous targets now return stable no-op workspace edits instead of accidental text rewrites.
- Formatting: added LSP `textDocument/formatting` support backed by external `c3fmt` over stdin/stdout, with configurable formatter path (`Formatting.c3fmt`, including directory auto-resolve to `build/c3fmt`) and config strategy (`Formatting.config`: explicit path, `":default:"`, or local `.c3fmt` discovery fallback).

## 0.4.0

- Support <* and *> comments (https://github.com/pherrymason/c3-lsp/pull/99) Credit to @PgBiel
- Show documentation on hover and completion (https://github.com/pherrymason/c3-lsp/pull/101). Credit to @PgBiel
- Support `distinct` types (https://github.com/pherrymason/c3-lsp/pull/107). Credit to @PgBiel
- Improve syntax highlighting in function information on hovering. Credit to @PgBiel
- Adds type information as well as other information to completions. Credit to @PgBiel
- Improve macro handling (https://github.com/pherrymason/c3-lsp/pull/103). Credit to @PgBiel

- Fixes some crashes while writing an inline struct member (https://github.com/pherrymason/c3-lsp/issues/97) and other language constructions.
- Optimizations to reduce CPU usage by 6-7x. Credit to @PgBiel https://github.com/pherrymason/c3-lsp/pull/99
- Fix parsing of non-type alias def (Credit to @PgBiel)
- Fix completion of enum and fault methods. https://github.com/pherrymason/c3-lsp/pull/111. Credit to @PgBiel


## 0.3.3
- Support named and anonymous sub structs.
- Fix clearing old diagnostics. (#89, #83, #71, #62)
- Fixed crash in some scenarios where no results were found.
- Fix crash when using with Helix editor (#87)

## 0.3.2

- Fix function unnamed argument types not being resolved correctly. Thanks to @insertt
- Fix not displaying correctly collection types on hover.

## 0.3.1

- Added compatibility with stdlib symbols for C3 0.6.2.
- Fixes diagnostic errors persisting after they were fixed (#62).
- C3 Version argument is deprecated. It will try to load last C3 supported version.
- Fallback to last C3 supported version if provided version is not supported.
- Binary name changed to `c3lsp` (before it was `c3-lsp`).

## 0.3

- LSP configuration per project: Place a c3lsp.json in your c3 project to customize LSP configuration. This will also allow to use new configuration settings without needing to wait for IDE extensions to be updated. See Configuration for details.
- Inspect stdlib. Use go to declaration/definition on stdlib symbols by configuring its path in c3lsp.json.
- Fixes diagnostics in Windows platform.

## 0.2.1

- Fixes once an error is detected, it does not disappear once the error is resolved (https://github.com/pherrymason/c3-lsp/issues/59)
- Fixes Go to definition / Declaration.

## 0.2

- New feature: error diagnostics. 
  In order to work, there are some requirements:  
  You will need last version of c3c (>=0.6.2) as some fixes had to be done there (Thanks @clerno !)  
  Either you have c3c in your PATH, or you use set its path with the new argument c3c-path.
- A new argument has been added to control the delay of diagnostics being run: diagnostics-delay. By default is 2 seconds.

## 0.1.0

- C3 language keywords are now suggested in TextDocumentCompletion operation. Thanks @nikpivkin for suggestion.
- Fix server crashing when writing a dot inside a string literal. #44 Thanks @nikpivkin for reporting.
- Fix server crashing when creating a new new untitled c3 file from VS Code without workspaces. Thanks @nikpivkin for fixing.
- Fix server crashing when not finding symbol. Thanks @tclesius for reporting.

## 0.0.8

- Fix Go to definition / Go to declaration on Windows.

## 0.0.7

- Include symbols for stdlib 0.6.1
- New argument --lang-version to specify which c3 version to use
- Improvements indexing stdlib symbols: Generic parameters were ignored.
- Improvements in resolving types referencing generic parameters.
- Go to declaration/definintion failed in Windows.

## 0.0.6
- New LSP feature: SignatureHelper
- Improved resolution of references to types inside indexed symbols (Examples: variable types, function argument and return types, definitions...)
- Support optional types.
- Include Stdlib v0.6 symbols.
- c3i files were ignored.
- Difficulties parsing symbols on files that contained bitstructs with defined ranges.
- Stdlib was not properly indexed, which resulted in stdlib symbols not resolved in some situations.
- Fix completion on stdlib struct methods.
- Clear obsolete indexed symbols as documents are modified.
- Remove indexed elements when a file is removed.
– Symbols were holding old filenames after files being renamed.

## 0.0.5
- Added supports for Enum methods and associative values.
- Parse type information of defs and struct members.
- Properly resolve access path when a def is traversed.
- PENDING: Properly resolve struct members when they include implicit module path
- Resolve partial module paths properly.
- Resolve stdlib symbols: This version includes stdlib symbols from c3 0.5.5.
- Fix passing arguments to server.

## 0.0.4
- Index function declarations. Useful for bindings for example.
- Improve Hover content by enabling syntax highlighting and including function arguments.
- Fix wrong module being resolved.
- Fix finding symbols on imported modules.

## 0.0.3
- TextDocumentCompletion:
  - Added support for fault and module names.
  - Improve completion of struct methods.
- Fix resolving wrong symbol when trying to find module name.
- Fix resolving wrong symbol when there are two or more with the same name in different scopes.
- Fix processing files without symbols.
- Added cli arguments to display configure server behavior.
- Send (optionally) crash reports to Sentry. Only stack traces will sent.

## 0.0.2
- Fix crash finding symbols.
- Generics.
- Added partial support for struct subtyping.

## 0.0.1
- Implements 
  - TextDocumentDeclaration
  - TextDocumentDefinition
  - TextDocumentHover
  - TextDocumentCompletion
