---
phase: 08-operations-repair-path-and-audit
verified: 2026-10-15T07:50:35Z
status: passed
score: 23/23 must-haves verified
covered_files: # 384 files — the 381 the previous pass declared, plus the three files of the dispatcher-guard round this pass re-adjudicated: internal/keys/keys.go (the keymap table the dispatcher is joined to, and one half of the join), internal/keys/keys_test.go (its own size pin, the 30 rows the join's anti-vacuity floors assume), and internal/tui/dispatch_test.go (the guard itself)
  - .planning/phases/08-operations-repair-path-and-audit/08-01-PLAN.md
  - .planning/phases/08-operations-repair-path-and-audit/08-01-SUMMARY.md
  - .planning/phases/08-operations-repair-path-and-audit/08-02-PLAN.md
  - .planning/phases/08-operations-repair-path-and-audit/08-02-SUMMARY.md
  - .planning/phases/08-operations-repair-path-and-audit/08-03-PLAN.md
  - .planning/phases/08-operations-repair-path-and-audit/08-03-SUMMARY.md
  - .planning/phases/08-operations-repair-path-and-audit/08-CONTEXT.md
  - RUNBOOK.md
  - cmd/spike/main.go
  - cmd/x-opsgen/main.go
  - cmd/x-terminal/main.go
  - internal/audit/audit.go
  - internal/audit/audit_test.go
  - internal/audit/guards.go
  - internal/audit/guards_test.go
  - internal/audit/leak.go
  - internal/audit/leak_test.go
  - internal/audit/readonly.go
  - internal/audit/readonly_test.go
  - internal/audit/source.go
  - internal/cli/audit.go
  - internal/cli/audit_test.go
  - internal/cli/cli.go
  - internal/cli/cli_test.go
  - internal/cli/doctor.go
  - internal/cli/doctor_test.go
  - internal/cli/leak_test.go
  - internal/cli/runbook_test.go
  - internal/cli/sync.go
  - internal/domain/saved.go
  - internal/domain/saved_test.go
  - internal/keys/keys.go
  - internal/keys/keys_test.go
  - internal/render/alias.go
  - internal/render/capture_panic_test.go
  - internal/render/capture_pty_test.go
  - internal/render/capture_tmux_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/golden_cover_corpus_test.go
  - internal/render/golden_test.go
  - internal/render/grammar.go
  - internal/render/help.go
  - internal/render/help_test.go
  - internal/render/profile.go
  - internal/render/saved.go
  - internal/render/testdata/TestGoldenFrames/bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_absent_rows_80.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_absent_rows_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_never_synced_80.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_synced_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_syncing_80.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_syncing_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_16.golden
  - internal/render/testdata/TestGoldenFrames/color_16_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_16_saved.golden
  - internal/render/testdata/TestGoldenFrames/color_16_saved_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_16_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_256.golden
  - internal/render/testdata/TestGoldenFrames/color_256_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_256_saved.golden
  - internal/render/testdata/TestGoldenFrames/color_256_saved_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_256_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_mono.golden
  - internal/render/testdata/TestGoldenFrames/color_mono_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_mono_saved.golden
  - internal/render/testdata/TestGoldenFrames/color_mono_saved_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_mono_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor_saved.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor_saved_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_offline_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_offline_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/empty_120.golden
  - internal/render/testdata/TestGoldenFrames/empty_80.golden
  - internal/render/testdata/TestGoldenFrames/expired_120.golden
  - internal/render/testdata/TestGoldenFrames/expired_80.golden
  - internal/render/testdata/TestGoldenFrames/help_120.golden
  - internal/render/testdata/TestGoldenFrames/help_80.golden
  - internal/render/testdata/TestGoldenFrames/likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/likes_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/likes_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/loading_80.golden
  - internal/render/testdata/TestGoldenFrames/longdetail_80.golden
  - internal/render/testdata/TestGoldenFrames/media_copied_80.golden
  - internal/render/testdata/TestGoldenFrames/media_copied_wl_80.golden
  - internal/render/testdata/TestGoldenFrames/media_fallback_120.golden
  - internal/render/testdata/TestGoldenFrames/media_fallback_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_no_opener_all_80.golden
  - internal/render/testdata/TestGoldenFrames/media_none_80.golden
  - internal/render/testdata/TestGoldenFrames/media_opened_80.golden
  - internal/render/testdata/TestGoldenFrames/media_wrap_exact_80.golden
  - internal/render/testdata/TestGoldenFrames/mirror_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/narrow_79.golden
  - internal/render/testdata/TestGoldenFrames/offline_120.golden
  - internal/render/testdata/TestGoldenFrames/offline_80.golden
  - internal/render/testdata/TestGoldenFrames/panic_120.golden
  - internal/render/testdata/TestGoldenFrames/panic_80.golden
  - internal/render/testdata/TestGoldenFrames/permalink_fallback_80.golden
  - internal/render/testdata/TestGoldenFrames/schemachange_120.golden
  - internal/render/testdata/TestGoldenFrames/schemachange_80.golden
  - internal/render/testdata/TestGoldenFrames/stalled_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_120.golden
  - internal/render/testdata/TestGoldenFrames/thread_200.golden
  - internal/render/testdata/TestGoldenFrames/thread_60.golden
  - internal/render/testdata/TestGoldenFrames/thread_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_adversarial_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_chaincut_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_16.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_256.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_mono.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_truecolor.golden
  - internal/render/testdata/TestGoldenFrames/thread_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_narrow_79.golden
  - internal/render/testdata/TestGoldenFrames/thread_partial_120.golden
  - internal/render/testdata/TestGoldenFrames/thread_partial_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_push_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_sel_ancestor_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_sel_quote_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_sel_reply_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_status_failure_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_status_noplus_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_status_plus_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_tombstone_focal_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_visibility_80.golden
  - internal/render/testdata/TestGoldenFrames/timeline_120.golden
  - internal/render/testdata/TestGoldenFrames/timeline_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_200.golden
  - internal/render/testdata/TestGoldenFrames/timeline_200_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_60.golden
  - internal/render/testdata/TestGoldenFrames/timeline_60_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_80.golden
  - internal/render/testdata/TestGoldenFrames/timeline_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_last_80.golden
  - internal/render/testdata/TestGoldenFrames/wrong_account_80.golden
  - internal/session/leak_test.go
  - internal/store/facts_test.go
  - internal/store/read.go
  - internal/store/schema_test.go
  - internal/tui/corpus_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/dispatch_test.go
  - internal/tui/golden_test.go
  - internal/tui/model.go
  - internal/tui/profile_test.go
  - internal/tui/testdata/TestGoldenFrames/profile_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_200.golden
  - internal/tui/testdata/TestGoldenFrames/profile_60.golden
  - internal/tui/testdata/TestGoldenFrames/profile_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_bare_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_bio_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_16.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_256.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_mono.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_truecolor.golden
  - internal/tui/testdata/TestGoldenFrames/profile_empty_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_loading_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_quote_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_repost_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_scrolled_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_tombstone_80.golden
  - internal/tui/testdata/TestGoldenFrames/prompt_120.golden
  - internal/tui/testdata/TestGoldenFrames/prompt_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_200.golden
  - internal/tui/testdata/TestGoldenFrames/search_60.golden
  - internal/tui/testdata/TestGoldenFrames/search_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_16.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_256.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_mono.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_truecolor.golden
  - internal/tui/testdata/TestGoldenFrames/search_empty_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_failure_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_stalled_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_top_80.golden
  - internal/tui/tui_test.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/xweb/graphql.go
  - internal/xweb/probe.go
  - internal/xweb/probe_test.go
  - internal/xweb/volatile/registry.go
  - internal/xweb/volatile/registry.json
covered_digest: "v3:sha256:c35823691742f34b1d53868e3861e046ce7bd77817b2d0dde97dcc4298cef5fd"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 23/23
  gaps_closed: # this pass closed nothing that was open: it re-adjudicated the phase after one narrow guard round and found every must-have still true
    - "no gap was open entering this pass; the round's own claims were falsified and none of them moved a truth of this phase"
  gaps_remaining: []
  regressions: [] # none: the round touched four files (internal/tui/update.go, internal/tui/dispatch_test.go, internal/keys/keys.go, internal/keys/keys_test.go) and moved no behaviour this phase rests on, as its own plants and the behaviour-preservation checks show; no golden byte moved (testdata aggregate sha256 18f992e7… identical to what the previous report recorded), and the full suite is the previous 1571 plus exactly the two new guard tests
  guard_round: # the one thing this pass was asked to check beyond the phase's own truths
    claim: "the press restructure is behaviour-preserving: each action has exactly one arm and the handled set is enumerable (handledActions + HandledActions())"
    verdict: "behaviour-preserving, and the round's own two plants fail as reported; the guard itself is NOT total over the dispatcher it names — see anti-pattern 11"
    evidence: "all 30 press arms are one-per-action and set-equal to handledActions (measured mechanically: 30 arms, 30 unique handled actions, zero in either difference); the flows that depend on press are green by their own names (keymap⇄decoder matcher, cover gate, prompt keys, panic gate, frame push/pop, back stack); the compiler refuses a bogus case label (update.go:334's planted arm, reproduced at :309)"
coincidental_reliance_items:
  - truth: "Over the covered corpus: no real handle/display name/product name"
    reason: fixture-only
    harden: "coveredSafeList/scrubCoverBody (internal/render/golden_cover_corpus_test.go:23,49) and coveredSavedPosts (:75) rewrite the product nouns out of the content before the frames are recorded, so the stored render corpus can never carry one: the arm's green over that corpus is established by the generator. The rule itself is armed, not vacuous — the productNouns table fires on a same-width substitution in a stored covered frame (cover_ci_thread_80 throttle->timeline, corpus_test.go:390, re-run this pass), status_test.go's plain/covers arm asserts the same table over the plain failure blocks, and the tui half has its own scanner (corpus_test.go:803) with its own plant proof (:843). The scan's word lists are narrower than the fixture's own vocabulary, which is anti-pattern 1."
deferred: # environment-gated measurements, not actionable in this environment and not unverified code
  - truth: "The panic path is re-verified with the probe harness on screens built since Phase 3, outside the multiplexer (bare Ghostty)"
    addressed_in: "not a later phase — run in a Ghostty window on the operator's host"
    evidence: "internal/render/capture_panic_test.go:290 skips with a printed reason and the command to run; this host has TERM_PROGRAM empty and TMUX unset. The multiplexer leg was measured again this pass on this host over the current frames: 62 labelled lines -> 0 per later screen (thread/bookmarks/likes/media_detail), with the no-clear control leaking 41; the run prints the per-run socket name it cleaned up."
  - truth: "A live `x-terminal doctor` / `doctor --probe` run against the real session"
    addressed_in: "not a later phase — run on the machine that holds the live session"
    evidence: "Running the verb on this host would spend real requests against the account, which the phase's own request discipline forbids for verification; the lines, exit codes and JSON twins are asserted hermetically (internal/cli/doctor_test.go:527 TestDoctorPresenceFactsTextAndJSON carries the per-outcome exit-code table, TestDoctorProbeOneRequestAndNoBody and TestDoctorLeavesMirrorBytesAndMtimeUnchanged both pass in the named run) and the report builders were read. A poisoned-HOME run this pass confirmed no test in internal/cli resolves the real session path (68 test names identical with and without the poison, module cache pinned, while a real doctor run demonstrably read it)."
---
# Phase 8: Operations, Repair Path and Audit Verification Report

**Phase Goal:** When the upstream breaks — a rotated query id, a broken signer, an expired cookie, a rate limit — the user can find out exactly what broke and repair it without touching code.
**Verified:** refreshed on the frozen tree after the guard round that restructured `internal/tui/update.go`'s `press` (one arm per action, an enumerable `handledActions`, and `internal/tui/dispatch_test.go`'s two-way join); previous report of this phase: 2026-10-15T06:23:41Z
**Status:** passed

This pass re-adjudicated the phase on the tree as it stands now, with one guard round in it: `press` was restructured so each action has exactly one arm, `handledActions` (with the exported `HandledActions()`) enumerates the set beside the dispatcher, and `internal/tui/dispatch_test.go` joins that set to the keymap in both directions plus a source sweep over `keys.go`, its rows and the shell's case labels. Nothing else in the tree moved: exactly four files outside `.git` and `.planning` are newer than the previous report's timestamp — `internal/tui/update.go`, `internal/tui/dispatch_test.go`, `internal/keys/keys.go` and `internal/keys/keys_test.go` — and the whole-tree manifest is byte-identical before and after the suite. Every claim the round makes was falsified by re-planting its pre-fix behaviour in a scratch copy (`/tmp/v8finaltree`, a byte copy of the repository, restored by `rsync --delete` after each plant) rather than taken on trust; the repository itself was never edited, and the suite's effect on it was measured rather than assumed (a whole-tree manifest of `mtime size path` over the 542 files outside `.git` and `.planning` — the previous pass's 541 plus `internal/tui/dispatch_test.go` — is identical before and after the full suite, and both `testdata` trees hash to the same aggregate `18f992e7…`, the digest the previous report recorded, so **no golden byte moved in this round**). The full suite reports the previous pass's 1571 passing tests plus exactly the guard's two new top-level tests (694 top-level + 879 nested = 1573, 0 failed, the same 8 by-design skips), so nothing the guard round touched removed an assertion. No git command was run; no file outside this report was edited. **The behaviour-preserving claim holds for the flows that depend on `press` (all green by name), but the guard is not the exhaustiveness proof its comment reads as: an arm whose action is also a case label elsewhere in the shell can be deleted with both directions green — anti-pattern 11 and adjudication (g).**

The headline result this pass is the auditor's own question about the guard: it is a *tree-level reachability claim* in the same family as the audit's `source` and `opnames` checks — one table joined to another across a seam nothing else joins — so it was adjudicated against the audit's remit and deliberately **left where it lives**, in `internal/tui`'s tests rather than in the audit's pinned check set. The audit's ten checks read the tree as bytes and import nothing of the application (`TestPackageImportsNeitherStoreNorXweb`; `go list -deps ./internal/audit` resolves to one module package), which is what lets it be the last thing standing when a package will not compile; the guard cannot be written that way — it *is* `keys.Rows()` against `handledActions`, a compiled join that only exists once the shell links. The sibling guard for the other seam says the same thing: `TestEveryVerbBodyIsReachableThroughCommands` (the AUTH-03 defect class the dispatch guard's own comment cites) lives in `internal/cli`'s tests, not in the audit, and the audit has never claimed a UI-consistency surface. So the guard stays a package test with its own plants, and this report records why it is not a missed check rather than a refused one.

## What the fix round (the previous pass's change) did, and the falsification that proves each

| # | Change (this phase's own artefacts) | Pre-fix behaviour re-planted | What the plant did | What the frozen tree does |
|---|------------------------------------|------------------------------|--------------------|---------------------------|
| 1 | The tenth check `opnames` (`internal/audit/readonly.go:161-200`): every operation name the classification table knows must be one the registry snapshot holds. Registered third (`audit.go:187`), pinned as the third name (`audit_test.go:99`) | `"ListLatestTweetsTimeline": "read",` put back into `opClass` (`readonly.go:31`) — the stale entry the check was written for | `opnames 1 files` + `internal/audit/readonly.go: the table classifies "ListLatestTweetsTimeline" as a read and the registry does not hold it: an entry for an operation this build cannot resolve`; `10 checks, 1 findings, 893 files inspected`, exit 1 | the table's 33 names are all in the snapshot's 107 operations; 0 findings, exit 0 |
| 2 | The pin: `opnames` deleted from `Checks()` | — (the removal is the plant) | `audit_test.go:123: the audit runs 9 checks [source readonly transport timers persistence distribution account boundaries leak], want the pinned 10 [source readonly opnames transport timers persistence distribution account boundaries leak]`, and the audit then prints `9 checks, 0 findings, 892 files inspected` | `TestCheckSetIsPinned` passes; `Checks()` returns the pinned 10 in the pinned order |
| 3 | The check cannot pass vacuously: an unread snapshot is a finding, an empty table is a finding, an undecodable snapshot is that check's own finding | `printf '{"operations": {}}' > internal/xweb/volatile/registry.json` (valid JSON, no operations) | 33 findings — one per table entry — `10 checks, 33 findings`, exit 1; and with `{broken json`: `internal/xweb/volatile/registry.json: does not decode: invalid character 'b' looking for beginning of object key string — the names it holds cannot be read, so the table is unchecked` beside the `readonly` check's own decode finding, 2 findings, exit 1 | the snapshot decodes; `opnames` reads 1 file and finds nothing |
| 4 | The stale entry is gone from the table: `ListLatestTweetsTimeline` survives only in `opnames`' doc comment (`readonly.go:170`) as the reason the check was written | — | the table holds 33 names, the snapshot 107 operations, and the intersection is total | 0 findings |
| 5 | `internal/store.FileError` marks the store's own file failures, and `internal/cli/feed.go:129-130`'s predicate is `errors.As(err, &file)` — a whitelist, so a shape-drifted answer and a store **write** failure no longer claim "mirror unreadable". The saved-frame work is phases 06/07's; what this phase rests on is that nothing moved | not re-planted here (another phase's truth); the cross-checks were run | `internal/store`'s suite passes including `TestTheFilesOwnReadFailuresCarryTheMark`; the audit's `store-source` surface still reads `internal/store` (0 findings); the doctor's mirror line and `TestDoctorLeavesMirrorBytesAndMtimeUnchanged` unchanged | all green |
| 6 | A capability can be cleared (`internal/screen/saved.go`), and `internal/tui/cover_e2e_test.go`'s "store cannot be read" scenario now drives a reachable state: `FailPage` plus the failing map (`:1322-1331`), where the frame's own read is what answers with the failure | — | reverting the saved pane's call sites to `render.Plain` fails `TestTheCoveredSavedPaneNamesNothingOfTheProduct` at `cover_e2e_test.go:1467`/`:1476` (14 state×width subtests / 216 findings), so the scenario is exercised, not decorative | the covered saved pane's arms pass; all 319 goldens byte-identical |
| 7 | Three documented promises made true: `keys.ByKeystroke`'s comment now says it is the tests' matcher with no production caller (`internal/keys/keys.go:215-219`); `internal/config`'s pins path is built from `volatile.PinsFileName` (`config.go:183`), with the constant pinned to `pins.json` so a rename is a migration (`config_test.go:740`); `domain.PostPage.Requested`'s doc says who writes it and that no read path reads it (`internal/domain/page.go:50-65`) | — | read, not planted: none is a truth of this phase, and the only one touching a file this report cites (the config destination) leaves the persistence check's printed destination set unchanged | the persistence promise prints the same six destinations and the same 10 reached call sites as the previous pass |

Line numbers above are re-measured on this tree where it moved since the previous report (the pin's failure line is `audit_test.go:123`, not `:119`; the `opnames` code sits at `readonly.go:161-200`; the request-builder allowlist is at `readonly.go:294-302`, not `:237-245`).

Re-run this pass, not merely kept: rows 1-4 (the `opnames` finding on the stale entry, its emptiness and undecodable arms, the pin failing by name at `audit_test.go:123`) were each re-planted in the scratch copy and reproduced exactly as written, and rows 6-7's cross-checks were re-run (the saved pane's `render.Plain` revert, the persistence destination set unchanged). Row 5 (`store.FileError`) is another phase's truth and was cross-checked, not re-planted — the store suite and `TestDoctorLeavesMirrorBytesAndMtimeUnchanged` are green.

## What the guard round (this pass's change) did, and the falsification that proves each

The round is a refactor of a live dispatcher, so the claim to check is behaviour-preservation, not new behaviour. Three facts were measured on the frozen tree before any plant: `press` (`internal/tui/update.go:227-329`) holds **30 arms, one per action**, and its set equals `handledActions` (`:1297-1309`, **30 unique** actions) with **zero** members in either difference (mechanically: parse the arms, parse the enumeration, diff both ways — `press-only: []`, `handled-only: []`). The arms that live in the *other* switches — the panic gate's `ActionPanic`/`ActionQuit` (`:233-238`) and `savedAction`'s `ActionBookmarks`/`ActionLikes` (`internal/tui/saved.go:248-250`) — are the same actions, not extra ones, which is exactly what makes the guard's mechanism blind (anti-pattern 11). `HandledActions()` is `slices.Clone(handledActions)` (`:1315`), and the guard asserts the copy (writing the returned slice does not move the shell's own).

The flows `press` exists for are green by their own names this pass: the keymap⇄decoder matcher (`TestTheKeymapMatchesWhatTheDecoderReports`), the gated actions under a cover (`TestCoverBreakingActionsGated`, `TestPanicIgnoresCoverBreakingKeys`), the prompt's own keys (`TestSearchPromptSpendsNothing`, `TestSearchSlashReseedsThePrompt`), the panic gate's keys (`TestPanicSwallowsKeys`, `TestPanicEntersInOneFrameAndLeavesExactly`, `TestBlurPanicsAndFocusDoesNotRestore`, `TestPanicRoundTripIsExact`, `TestPanicEntryOrderAndBranches`), and the frame pushes/pops (`TestTheBackStackRestoresTheFrame`, `TestEnterOpensAConversationWithLoadingThenPage`, `TestEnterOnAQuotedPostPushesTheQuote`, `TestQRestoresTheFrameWithZeroThreadRequests`, `TestTheWholeSurfaceUnderACover`) — plus all 319 goldens byte-identical. The compiler half of the claim reproduces too.

| # | Change (the guard round's own artefacts) | Plant re-run | What the plant did | What the frozen tree does |
|---|------------------------------------------|--------------|--------------------|---------------------------|
| 1 | Each action has exactly one arm in `press`; `handledActions` enumerates the set; `HandledActions()` exports a copy | a **31st row** with a fresh action (`ActionZzPlantNew`, key `Z`) added to `keys.go`'s table and declared beside the others — the author's first plant | `dispatch_test.go:63: the keymap binds "zz-plant-new" (Z) and press has no arm for it: the key is a documented no-op`; `:79: the table holds 31 rows and handledActions names 30 actions`; and the sweep: `:128 keys.go declares ActionZzPlantNew and handledActions does not name it`, `:159 … is no case label in the shell` | 30 rows, 30 handled actions, the join green |
| 2 | The same, direction two | **a handler with no row** — a fresh action declared, an arm added to `press`, an entry added to `handledActions`, and no keymap row (the author's second plant) | `dispatch_test.go:72: press handles "zz-plant-no-row" and no row of the keymap binds it: the arm is unreachable`; `:79 the table holds 30 rows and handledActions names 31 actions`; `:125 keys.go declares ActionZzPlantNoRow ("zz-plant-no-row") and no row binds it` | green |
| 3 | The compiler is the guard's floor for case labels (the property `update.go:331-337`'s comment leans on) | `case keys.ActionThisPackageDeclaresNoSuchAction: return m.enterPanic()` planted at `update.go:309` | `go build ./...` → `internal/tui/update.go:309:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction` | 30 arms, each a declared constant |
| 4 | The join is exhaustive over the dispatcher, i.e. an arm deleted from `press` fails it (the guard's stated purpose) | `case keys.ActionSearchOrder:` + its body deleted from the main switch (an action whose only case label is that arm) | `dispatch_test.go:159: keys.ActionSearchOrder is no case label in the shell: nothing dispatches it` — **caught** | — |
| 5 | The same, and this is where the claim breaks down | the main-switch arm (whole arm, body included) deleted for **all 30** actions in turn, one probe per action in a 22-package scratch copy (`/tmp/v8t3` for the whole-suite counts) | **15 of 30 arms are caught** (Open, OpenAuthor, Refresh, SwitchList, SearchPosts, SearchPeople, SearchOrder, Help, Back, Cover, NextCover, Cover1-4); **15 pass both directions** — exactly the actions whose constant is also a case label elsewhere in the package: `Next`/`Prev`/`HalfDown`/`HalfUp`/`PageDown`/`PageUp`/`First`/`Last` in `move` (`:381-397`), `Media`/`CopyLink`/`OpenPost` in `coverBreaking` (`:342-346`), `Quit`/`Panic` in the panic gate (`:233-238`), `Bookmarks`/`Likes` in `savedAction` (`saved.go:248-250`). See anti-pattern 11 and adjudication (g) | the behavioural suite catches behaviour failures for 12 of the 15 (Next 4, Prev 8, PageDown 1, First 1, Last 19, Quit 2, Panic 9, Media 8, CopyLink 2, OpenPost 1, Bookmarks 9, Likes 3) and catches **none** of `ActionHalfDown`, `ActionHalfUp`, `ActionPageUp` — 0 failures across all 22 packages, so those three keys become silent no-ops with nothing in the tree objecting |

One more plant, unrelated to the guard but part of the round's blast radius: deleting the `press` arm for `ActionLikes` (the same deletion as row 5) still fails `TestSavedFrameIssuesNoSourceRequest` (`tui_test.go:1444: likes top kind = 0`) and `TestLikesUnavailableLeavesOthersAlone` (`:1618`) — the covered saved pane's own arms are what hold that flow, independent of the guard.

## What the amendment round (the previous pass) did, re-run this pass

The previous pass's table is kept, with each row re-run in kind this pass (the plants are this pass's own, in `/tmp/v8finaltree`). All twelve reproduced:

| # | Change (this phase's own artefacts) | This pass's re-run |
|---|------------------------------------|--------------------|
| 1 | The audit's first check is `source` (`internal/audit/source.go`, 112 lines): every non-test Go file the audit reads must parse | `func broken( {` appended to the real `internal/xsource/home.go` → `source: 99 files, 1 findings` + `internal/xsource/home.go:123: cannot parse: expected ')', found '{'`; every Go scan's count drops by one (readonly 94→93, transport/timers/persistence/account 99→98; boundaries stays 99), the total drops by the same five |
| 2 | Remove `source` from `Checks()` and the old silent pass returns | an unparseable `cmd/x-opsgen/main.go` with an `os.WriteFile` behind the syntax error: `9 checks, 0 findings, 789 files inspected`, exit **0** — the pre-amendment silent pass reproduced exactly |
| 3 | The same removal against the pin fails by name | `audit_test.go:123` names the 9 that ran and the 10 that are pinned |
| 4 | A registry JSON that does not decode is a finding, not an empty name set | `{broken json` → `readonly` finding naming the file (and now `opnames`' own, above), exit 1 |
| 5 | A tree missing `internal/render/copy.go` or the `internal/store` sources is a finding, not one surface fewer | `rm internal/render/copy.go` → `no media-notice-constants surface: …` (`leak` counts 103); `rm internal/store/*.go` → `no store-source surface: …` (`leak` counts 100) |
| 6 | The covered likes frame prints the cover's own hedge, pinned by 8 frames and the corpus arm | the covered branch reverted to the plain line → `saved_test.go:80` on all four styles (`the servicelog style's likes line is " no likes, or the list is no longer served", want the cover's own hedge …`) and `TestGoldenFrames` failing on exactly the 8 `cover_*_likes_hedged_{80,120}` frames |
| 7 | Both corpora are scanned for the plain vocabulary, each with an in-package plant proof | the permanent arms pass (`TestCoveredLeakScanCatchesAPlant` over `cover_gitlog_thread_80` and `cover_gitlog_bookmarks_80`, `TestTheCoveredTUIFrameLeakScanCatchesAPlant`), and a same-width `throttle`→`timeline` substitution in the committed `cover_ci_thread_80.golden` fails the render scanner at `corpus_test.go:390` naming the frame |
| 8 | `frameKinds()` is joined to the per-kind check table both ways | a sixth kind appended (`frameKind(99)`) → `cover_e2e_test.go:739` and `:743` (5 rows vs 6 kinds) |
| 9 | The four covered notices have frames: 32 stored, with a totality arm | deleting `cover_gitlog_sync_notice_120.golden` is caught twice: `corpus_test.go:2266` and `golden_test.go:352` |
| 10 | A covered profile names one account consistently (`internal/tui/view.go:447-455` `profileAccount`) | both pre-amendment shapes fail: naming the opened handle under a cover → `cover_e2e_test.go:835`; naming nothing even after the header → `:844`/`:851` |
| 11 | Every cover stamp sets `MonthYear` (`cover.go:197/203/207/210`), so each cover's fourth facts clause renders | removing `coverGitStamp.MonthYear` fails `internal/tui`'s `TestGoldenFrames/profile_gitlog_80` and `/profile_gitlog_120`; the render corpus holds no profile frames |
| 12 | The tmux capture harness names its socket per run and cleans up | the socket fixed to `xterm-golden` → `capture_tmux_test.go:166: the harness socket "xterm-golden" does not name this run (pid …)`; the multiplexer panic leg then measured 62 → 0 per later screen with the control leaking 41 |

The 9 dead `copy.go` constants stay gone (all ten names enumerated in 07-VERIFICATION:108 are absent from `internal/` and `cmd/`); one dead constant remains (`CopyNoticeMediaOne`, anti-pattern 9).

## Goal Achievement

### Observable Truths

Plan 01 (SEC-03, `internal/audit` + `x-terminal audit`):

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `audit` prints one line per check with its inspected-file count and every finding as file:line, exits non-zero on any finding, and a zero-file inspection is itself a finding | ✓ VERIFIED | Re-run live this pass: `go run ./cmd/x-terminal audit` → 10 check lines each carrying `N files` (`source 99`, `readonly 94` — the audit package's own five non-test Go files are out of its own scan, `opnames 1`, `transport/timers/persistence/account/boundaries 99`, `distribution/leak 102` — the last two count the whole loaded tree: 99 Go sources + `registry.json` + `go.mod` + `go.sum`), closing line `10 checks, 0 findings, 893 files inspected`, exit 0. The arithmetic is honest and additive: 99+94+1+99+99+99+102+99+99+102 = 893, and the tool's own `--json` agrees (`{checks:[{name,promise,files,findings}]}`, `findings: 0`, `files: 893`, the same 10 names in the same order). Removing one check drops exactly one check and one file (plant 2 above: `9 checks, … 892 files`); adding a file raises the total by its own count in every check that reads it (a root `Dockerfile` → `10 checks, 1 findings, 895 files inspected`, exit 1); `TestEmptyTreeIsAFinding` asserts the zero-file arm over every check |
| 2 | The read-only check classifies every spelled operation name and fails on unclassified, write-classified, and empty sets | ✓ VERIFIED | `readonly.go` 337 lines, table (33 names) + three arms; `readonly_test.go` covers all four cases over in-memory trees. Re-run plant this round: `const zzPlantOpName = "CreateTweet"` appended to `internal/xsource/home.go` → `internal/xsource/home.go:123: write operation name "CreateTweet" is spelled in source`, 1 finding, exit 1. New this round: the name set the check reads can no longer be an empty set in silence (`readonly.go:210-213`), and the table's own promissory half is now the `opnames` check (row 1 of the fix-round table) |
| 3 | No non-GET method is constructed anywhere, and no package outside `internal/xweb` builds an `*http.Request` | ✓ VERIFIED | Re-run plant: a new `internal/cli/zzplant.go` with `http.NewRequest(http.MethodPost, …)` → three findings at that one call site (`builds an *http.Request outside the GET request-builder packages`, `request method POST is not GET`, `http.MethodPost is a non-GET method constant`), exit 1. Deviation, unchanged: the builder allowlist is `internal/xweb` + `internal/netcheck` + `internal/opsgen` (`readonly.go:294-302`), two pre-existing GET-only builders; the printed promise is `no mutation is expressible over the wire`, which the method arm enforces for all three |
| 4 | No timer or sleep exists outside the limiter's inter-request floor | ✓ VERIFIED | `guards.go`; the tree's single exemption is printed in the report line (`… (exemption internal/xweb/limiter.go:66 sleepContext)`). Re-run plant: `time.NewTicker` in the planted `internal/cli/zzplant.go` → `banned timer call time.NewTicker`, exit 1; `TestTimersFailOnTicker` and `TestTimersFailWhenExemptionMissing` pass in this pass's named run |
| 5 | Every write-capable call site resolves to a declared destination; a fifth one fails the check | ✓ VERIFIED | The live promise names six destinations and every call site that reached each (`spike-recording@cmd/spike/main.go:176, notice@internal/cli/notice.go:64, config@internal/config/config.go:465, config@internal/config/config.go:524, saved@internal/store/read.go:198, saved@internal/store/schema.go:50, saved@internal/store/schema.go:56, pins@internal/xweb/volatile/pins.go:81, registry@internal/xweb/volatile/registry.go:399, registry@internal/xweb/volatile/registry.go:410`); re-run plants: a write to an undeclared path in a new `internal/cli/zzplant.go` → `write-capable call os.WriteFile resolves to no declared destination: a write path nobody declared is a persistence file nobody named`, exit 1; renaming `config` in the destination table fails three arms including the set pin (`guards_test.go:160: declared destinations are [confIg pins notice saved registry spike-recording], want [config pins notice saved registry spike-recording]`) |
| 6 | No distribution artifact and no hosted mode | ✓ VERIFIED | `guards.go` distribution check; re-run plant: a root `Dockerfile` → `Dockerfile: distribution artifact "Dockerfile" exists`, 1 finding, `895 files inspected` (the plant is counted twice: it is in both whole-tree counts), exit 1; the `net.Listen`/`http.Server`/bind-address arms have their own tests |
| 7 | Exactly one session-path resolution and one credential pair, with no per-account flag or config key | ✓ VERIFIED | Re-run plant: `var zzPlantCfg = Config{SessionPath: "/tmp/zz-session"}` appended to `internal/config/config.go` → `internal/config/config.go:555: second session-path resolution: session-path resolution (Config.SessionPath)`, 1 finding, exit 1 (the line moved from `:550` because this fix round edited the file's pins path); `TestAccountFailsSecondSessionPath` and `TestAccountFailsZeroSessionPath` pass in this pass's named run |
| 8 | The cookie sweep runs over the doctor's text and JSON output, the sync output, the audit's own report, the media notices, the store's error strings and a recovered panic's output, and finds the fixture cookie in none of them | ✓ VERIFIED | Re-run this pass: `searched 7 real surfaces for two credentials; both absent` (`internal/cli/leak_test.go:52`) over surfaces built by their own producers — the doctor's text and JSON reports, the sync report over the fake fetcher, `render.NoticeLine` notices, a `store.ReadFacts` error from a file that is not a database, `audit.Run(moduleRoot)`'s report and a panic recovered in the process — handed to `audit.Sweep`, the same function the audit's own leak check runs. The permanent planted arm (`:54-66`) requires the same sweep to fail when the fixture token is planted into the sync surface, so the pass is evidence rather than a search that cannot fail |
| 9 | Every check has a synthetic violation test and a real-tree planted-violation observation recorded in the SUMMARY | ✓ VERIFIED | All **ten** checks have synthetic arms — the source check's three, the `opnames` check's five (`readonly_test.go:95`, `:115`, `:130`, `:138`, `:147`: an entry the snapshot lacks, the real table against the real snapshot, a tree with no registry, an undecodable registry, an empty table), the rest in `guards_test.go`/`readonly_test.go`/`leak_test.go`; all green in this pass's named runs. The real-tree observations live in this report's two plant tables (the previous pass's and this round's), each re-run. Documentation drift, ℹ️ Info, widened this round: `08-01-SUMMARY.md` still records plan 01's own provenance — `8 checks, 0 findings, 782 files` (`:87-88`) and "Eight checks" (`:110`) — which was true when plan 01 ran and is now two checks and 111 files short of the tree; the SUMMARY is not updated and this report does not pretend it is; the guard round's two plants are recorded in this pass's guard table (row 1 and row 2) |

Plan 02 (OPS-03, the extended doctor):

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 10 | `doctor` gains lines, never a restructuring: mirror, signer, media openers, and a runbook pointer on every failing line; text and JSON carry the same facts | ✓ VERIFIED | `internal/cli/doctor.go` adds the labelled `mirror`/`signer` lines, appends the media facts to the config line and `RunbookAnchor(checkState(r))` to every failing check, in text and as `runbook` in JSON; stage order, the 90 s backstop and the `--show-notice`/`--refresh-ids`/`--pin` bodies are unchanged; `TestDoctorFailingCheckCarriesRunbookPointer` (`doctor_test.go:668`), `TestDoctorMediaLineHasBothDispatches` (`:505`) and `TestDoctorPresenceFactsTextAndJSON` (`:527`) pass in this pass's runs. This round's fix touched none of these files |
| 11 | The mirror line is read through the store's read-only path, presence facts only, and a missing mirror is not a failure | ✓ VERIFIED | `internal/store/read.go` opens `mode=ro&_pragma=query_only(1)`, never migrates; `TestDoctorLeavesMirrorBytesAndMtimeUnchanged` (`doctor_test.go:357`) re-run this pass and passing; `TestDoctorMirrorLineShapes` (`:262`) asserts four line shapes; missing → exit 0 with an empty pointer (`TestRunbookAnchorTableIsTotal`, `:471`). The fix round's `FileError` mark rides on the same read path and left this unchanged (re-checked this pass) |
| 12 | `doctor --probe <operation>` issues exactly one request through the same choke point and classifier, prints the state word plus rate-limit headers and never a body; an unknown operation is a usage error naming the ones it can build | ✓ VERIFIED | `internal/xweb/probe.go` (74 lines, no body field; the request goes through `Client.Do` → `Limiter.Do`); `TestDoctorProbeOneRequestAndNoBody` (`doctor_test.go:395`) re-run this pass and passing (exactly one request including the 429 case, and a planted body marker never reaches the `Probe` or its string form); `TestDoctorProbeUnknownOpIsUsage` (`:441`) asserts the usage error |
| 13 | The signer line is a presence fact measured in the same run | ✓ VERIFIED | `graphql.go:65` `SignerName()` returns `none` under the noop signer with the measurement in its doc comment; `doctor.go` adds the clause and the runbook pointer only on the signer-or-auth verdict; the exact line is asserted in `doctor_test.go` |
| 14 | Nothing the doctor prints is a token value, in text and JSON alike, and the new lines are covered by the sweep | ✓ VERIFIED | Presence-only lines confirmed by reading; `TestDoctorPresenceFactsTextAndJSON` fails with `report leaked credential material` when a token reaches either report, for the raw and both base64 forms, with a control string that proves the search can find it; the doctor's text and JSON are two of the seven real surfaces truth 8's sweep covers (re-run this pass). The `internal/cli` tests are hermetic: a poisoned-HOME run this pass (`HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME` at an empty directory, module cache pinned) listed the **same 68 test names** and passed, while a real doctor run demonstrably reads the session path |
| 15 | The doctor still exits non-zero on a failed link, and a skipped session is still not a failure | ✓ VERIFIED | `doctor_test.go:527`'s `TestDoctorPresenceFactsTextAndJSON` asserts the exit code per stubbed outcome (failing link non-zero, skipped session zero, missing mirror zero, newer/unreadable mirror non-zero) and the JSON `ok` field; `TestDoctorRefusesToPrintTheNoticeIntoAJSONReport` (`:103`) pins the usage code |

Plan 03 (OPS-06, the runbook and the covered re-audit):

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 16 | `RUNBOOK.md` exists with a section per OPS-06 case plus the mirror and the likes endpoint | ✓ VERIFIED | `RUNBOOK.md` at the root, 150 lines, 15 `## ` headings (14 sections plus `## Request discipline`); anchors match every failing state; the doctor's pointer table has 22 rows (12 xweb + 7 netcheck + 3 mirror) and every one resolves to a heading |
| 17 | Every section has the four parts, symptom quoted from the classifier's own words | ✓ VERIFIED | `TestRunbook`'s `four parts and verbatim symptoms` subtest asserts non-empty parts, their order and a verbatim classifier/state word; re-run plant this pass: truncating `## Soft ban` to its Symptom line failed it at `runbook_test.go:173` (`section "Soft ban" does not carry the four parts in order`), with the missing-part lines at `:169` and the exercisability arm at `:148`/`:153` |
| 18 | Every entry is exercisable: verification names a built command | ✓ VERIFIED | Every section's `**Verify.**` names a built command (14 `**Verify.**` lines, logged one per section at `runbook_test.go:150`); re-run plant this pass: replacing one with `look at the UI and see whether it seems fine` → `runbook_test.go:148: section "Soft ban" Verify names no built command: "look at the UI and see whether it seems fine"` |
| 19 | The request discipline is stated near the top and repeated in the 429 and soft-ban sections | ✓ VERIFIED | `RUNBOOK.md:6-8` plus the repeated rule in both sections; asserted by the `request discipline` subtest (three clauses), which failed this pass when `## Soft ban` was renamed (`runbook_test.go:203` for `one request`, `no retry loop`, `second diagnosis`) — and only then, so the arm reads the section rather than the file |
| 20 | The runbook cannot rot silently: a state without a section fails the build | ✓ VERIFIED | `runbook_test.go`'s coverage table is total over `failingXwebOutcomes()` (12) and `failingNetcheckOutcomes()` (7); `TestRunbookAnchorTableIsTotal` (`doctor_test.go:471`) asserts the doctor's table has a row and a non-empty pointer for each. Plants re-run this pass: the `soft ban` row deleted from the doctor's table → `doctor_test.go:486: missing xweb row "soft ban"` (+ `:489` empty pointer); the pointer retargeted at `RUNBOOK.md#soft-ban-gone` → `runbook_test.go:118: doctor pointer "RUNBOOK.md#soft-ban-gone" for "soft ban" has no heading`; the `## Soft ban` heading renamed → `runbook_test.go:82` (state maps to a heading that is not there), `:118`, `:203` and `:221` (orphan section) |
| 21 | The golden corpus is every screen x every cover style x both widths, and a screen with no covered frame is a missing file | ✓ VERIFIED | **263 stored frames** in `internal/render/testdata/TestGoldenFrames` — 154 `cover_*` (114 before the amendment round plus the 40 it added: 8 covered likes-hedged + 32 covered notices), 16 `media_gated_*`, 64 `_tmux` twins, 93 others — and 56 in `internal/tui/testdata/TestGoldenFrames` (24 covered), **319 combined**, generated as cross-products (`golden_cover_corpus_test.go`, `internal/tui/golden_test.go:337`). Re-run plants this pass: deleting `cover_ci_thread_80.golden` → `golden_test.go:352: open testdata/TestGoldenFrames/cover_ci_thread_80.golden: no such file or directory`; deleting the new `cover_gitlog_sync_notice_120.golden` → `corpus_test.go:2266` and `golden_test.go:352`; the tui corpus's floor arm requires at least 24 covered frames (`corpus_test.go:806`) |
| 22 | Over that corpus: no real handle/display name/product name; help output byte-identical across states; the three cover-breaking actions still gated; the covered refusal and help rows present | ✓ VERIFIED (coincidental-reliance) | Re-verified over the 319 frame bytes. (i) A same-width substitution in a committed covered frame (`cover_ci_thread_80.golden`: `throttle` → `timeline`) still fails the product-noun arm exactly: `corpus_test.go:390: the stored frame cover_ci_thread_80 prints "timeline", the product's own word (from CopyEmptyHeadingFollowing)`. (ii) A user row planted into the committed `search_80.golden` (`\n alice            Alice`) fails the reverse-homogeneity scanner: tui `corpus_test.go:205: search_80 posts frame carries a user row: "alice            Alice"` — with the control arm (`:187-199`) proving the same shape test finds four such rows in `search_people_80`. (iii) The tui leak scan over all 24 covered frames (`corpus_test.go:803`) and both in-package plant proofs pass. (iv) The covered saved pane's 7 states x 2 widths x 4 styles pass, and reverting its call sites to `render.Plain` fails 14 subtests / 216 findings at `cover_e2e_test.go:1467`/`:1476` |
| 23 | The panic path is re-verified with the probe harness on screens built since Phase 3, both inside and outside the multiplexer | ✓ VERIFIED (multiplexer leg measured here; bare-Ghostty leg deferred, see Deferred) | Re-measured this pass on this host: `TestCapturePanic/the_multiplexer's_history/the_shipped_program` → `capture_panic_test.go:164: the pane %0 held 62 labelled lines before and none in its screen or history after the entry`; the no-clear control → `:195: the control leaked: the pane holds 41 labelled lines with the clear disabled`; each post-Phase-3 covered screen → `:278: screen cover_gitlog_<thread,bookmarks,likes,media_detail>_80: before=62 labelled lines, after=0; history cleared after covered content was on screen`. The run exits 0; `/the_window` skips by design with the command printed (`:290`) |

**Score:** 23/23 truths verified (0 failed, 0 present-behavior-unverified, 1 advisory coincidental-reliance, 1 environment-gated deferral)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/audit/audit.go` | `File`/`Tree`/`Load`/`Finding`/`Report`/`Check`/`Checks`/`Run`, panic recovery, no store/xweb/cli import | ✓ EXISTS + SUBSTANTIVE | 253 lines; `Load` walks 102 files (99 Go sources, the registry JSON, `go.mod`+`go.sum`) and skips `testdata`/`.git`/`.planning`/`.agents`/dotfiles; `Checks()` returns **10** checks with `source` first and `opnames` third (`:183-196`); `TestPackageImportsNeitherStoreNorXweb` passes |
| `internal/audit/source.go` | the load-level honesty check: parse every non-test Go file, name the ones that do not parse, and give every Go scan the parsed set | ✓ EXISTS + SUBSTANTIVE | 112 lines; `checkSource` counts all Go files as read and reports one finding per unparseable file (path, parser message, stop line); `parseGoSources` (`:55`) is what the scans count; the pre-amendment silent pass is re-falsified (fix-round table row 2) |
| `internal/audit/readonly.go` | operation-name check (3 arms) + the classification table's own `opnames` check + transport check | ✓ EXISTS + SUBSTANTIVE | 337 lines; `opClass` holds 33 names (`:27-64`), `checkOpNames` at `:161-200`, `registryNames`/`registryJSONFile` at `:128-147`, the request-builder allowlist at `:294-302`; the registry's decode failure is a finding at each check that reads it |
| `internal/audit/audit_test.go` | the check set written out by hand and asserted in order | ✓ EXISTS + SUBSTANTIVE | 297 lines; `auditCheckNames` (**10** names, `source` first, `opnames` third, `:93-107`) + `TestCheckSetIsPinned` (`:120-140`) + `TestRunRealTree` + the three source arms; deleting a check fails by name (re-run, `:123`) |
| `internal/audit/readonly_test.go` | the `opnames` synthetic arms and the readonly/transport arms | ✓ EXISTS + SUBSTANTIVE | 210 lines; five `opnames` arms (`:95`, `:115`, `:130`, `:138`, `:147`), including the real table against the real snapshot through `checkOpNames` alone |
| `internal/audit/guards.go` | timers, persistence, distribution, account, boundaries | ✓ EXISTS + SUBSTANTIVE | 675 lines; persistence declares six destinations, prints all six plus every reached call site, scans every non-test file, fails a declared destination nothing writes to; every Go scan in the package counts only parsed files |
| `internal/audit/guards_test.go` | the destination set pinned by name and by printed text | ✓ EXISTS + SUBSTANTIVE | 285 lines; the rename plant fails the set pin by name (re-run, `:160`) |
| `internal/audit/leak.go` | cookie sweep over the named surfaces | ✓ EXISTS + SUBSTANTIVE | 208 lines; `Sweep` exported; a tree missing the copy table or the store sources is a finding (`:48-62`, re-run) |
| `internal/audit/testdata/` | — | ✓ REMOVED as designed | the directory does not exist and nothing in the package reads one; the only `testdata` occurrence is `Load`'s `SkipDir` (`audit.go:132`) |
| `internal/cli/audit.go` | `audit` verb, `--json`, findings as file:line, non-zero exit, bounded report | ✓ EXISTS + SUBSTANTIVE | 132 lines; verified live: 10 checks / 0 findings / 893 files, exit 0; JSON `{checks:[{name,promise,files,findings}], findings, files}`; a finding's `detail` is clipped at 200 chars |
| `internal/cli/cli.go` | exactly one new command-table row, and it must be reachable | ✓ EXISTS + SUBSTANTIVE | `audit    check the tree against the promises this build makes`; `TestAuditListedInUsage` and `TestEveryVerbBodyIsReachableThroughCommands` pass in this pass's run |
| `internal/cli/leak_test.go` | the CLI's own output surfaces swept for real | ✓ EXISTS + SUBSTANTIVE | 163 lines; seven surfaces from the producers themselves, asserted non-empty, plus a permanent planted arm that must fail on the same surfaces (re-run: token into the sync surface → the arm at `:54-66`) |
| `internal/session/leak_test.go` | the Session/Secret sinks | ✓ EXISTS + SUBSTANTIVE | 196 lines; unchanged |
| `internal/cli/runbook_test.go` | coverage, anchors, exercisability, four parts, orphan arm | ✓ EXISTS + SUBSTANTIVE | 382 lines; all arms demonstrated failing by this pass's plants (`:82`,`:118`,`:148`,`:153`,`:169`,`:173`,`:203`,`:221`) |
| `RUNBOOK.md` | 14 four-part sections + request discipline | ✓ EXISTS + SUBSTANTIVE | 150 lines, 15 `## ` headings, 14 `**Verify.**` lines |
| `internal/tui/corpus_test.go` | corpus scanners + planted-violation arms | ✓ EXISTS + SUBSTANTIVE | 905 lines; the profile cases are asserted over the stored bytes and each planted arm names the frame it lost (re-run: `:205`); the covered-frame scanner (`:803`) runs over all 24 covered frames with the aliases masked and a 24-frame floor (`:806`) |
| `internal/tui/testdata/TestGoldenFrames/` | the covered cross-product and the restored cases | ✓ EXISTS + SUBSTANTIVE | 56 frames; the 8 covered profile frames carry the covers' own four facts clauses including the since clause (`Since: Mar 2011`, `since=2011-03`, `[since 2011-03]`, `since Mar 2011`, read out of the committed bytes); 24 covered frames |
| `internal/tui/model.go` | the shell's own frame enumeration | ✓ EXISTS + SUBSTANTIVE | 716 lines; `frameKinds()` at `:196-198` returns the five kinds; joined to the covered check table both ways by `TestEveryFrameKindIsCheckedUnderACover` (a sixth kind fails it, re-run) |
| `internal/tui/cover_e2e_test.go` | the covered-surface arms on the shipped path | ✓ EXISTS + SUBSTANTIVE | 1513 lines; `savedFrameStates()` × 2 widths × 4 styles (the plain reader as its own arm; re-run: the `render.Plain` revert fails 14 state×width subtests / 216 findings at `:1467`/`:1476`); `frameKindChecks` + the join test (`:730-799`); `TestTheCoveredProfileNamesOneAccount` (`:800`) with both falsifications re-run; the "store cannot be read" state now drives a failing read plus the failing map (`:1322-1331`) |
| `internal/render/help.go` | the overlay cut to the pane's row count, the cut named | ✓ EXISTS + SUBSTANTIVE | 63 lines; the cut row is still at `help_80.golden:23` and `help_120.golden:23`, and no covered help frame carries it |
| `internal/render/testdata/TestGoldenFrames/` | every screen x cover x width | ✓ EXISTS + SUBSTANTIVE | 263 frames (154 `cover_*` including the 40 the amendment round added, 16 `media_gated_*`, 64 `_tmux` twins); every frame equals its generator byte for byte (the suite passes with no `-update`, and the aggregate hash is unchanged from the previous report) |
| `internal/render/saved.go` | the three state functions taking the pane's grammar | ✓ EXISTS + SUBSTANTIVE | 258 lines; `CapabilityBlock`/`SavedEmptyState`/`StateBlockWithMirror` take `Grammar` first; reverting the call sites fails the shipped-path arm (14 state×width subtests / 216 findings, re-run); the covered likes branch prints the cover's hedge (`:175-178`) |
| `internal/render/cover.go`, `grammar.go`, `profile.go` | the covers' own facts-row vocabulary | ✓ EXISTS + SUBSTANTIVE | the four grammars carry `AccountFacts` (`cover.go:259/311/357/408`); every cover stamp sets `MonthYear` (`cover.go:197/203/207/210`), so the `{joined}` clause renders in all four styles (the 8 covered tui profile frames); removing one stamp's `MonthYear` fails that style's golden frames |
| `internal/render/alias.go` | the cover alias dictionary | ✓ EXISTS + SUBSTANTIVE | `mirror` is still one of `coverRoles` (`:138`) — see anti-pattern 4; the tui scanner masks each frame's own aliases, which is the arm that keeps that latent false positive from firing on `mirror-merge-65` |
| `internal/tui/view.go`, `update.go` | the covered saved pane's grammar, `g`'s restore, `profileAccount` | ✓ EXISTS + SUBSTANTIVE | `view.go:412/417/423` pass `m.grammar()`; `update.go:376-383` clears `Moved` and `Anchor` together for `ActionFirst`; `view.go:447-455` `profileAccount` — header's account once answered, opened handle only for the plain reader, nothing under a cover |
| `internal/render/capture_tmux_test.go` | the multiplexer capture, one socket per run | ✓ EXISTS + SUBSTANTIVE | 401 lines; the socket names the run (`:162`); the fixed-name plant fails `:166` (and is invisible to `-run TestCapture` — anti-pattern 7); the harness kills its own server and removes its socket |
| `internal/render/capture_panic_test.go`, `capture_pty_test.go` | panic legs on the later screens | ✓ EXISTS + SUBSTANTIVE | tmux leg re-measured this pass on this host (62 → 0 per screen, control 41); Ghostty leg skips by design with the command printed (`:290`) |
| `internal/domain/saved.go`, `internal/store/read.go`, `internal/store/{facts,schema}_test.go` | `MirrorFacts` + the read-only path + the allowlist | ✓ EXISTS + SUBSTANTIVE | `MirrorFacts` has a closed field set with no per-row instant; `ReadFacts` uses `mode=ro` + `query_only` and never migrates (mtime test re-run and passing); `read.go` now also carries `FileError` (`:37`), returned by the store's own open/page/status failures, and `schema_test.go`'s allowlist names it (`:71`) — the fix round's mark, with `TestTheFilesOwnReadFailuresCarryTheMark` green |
| `internal/xweb/probe.go`, `internal/xweb/graphql.go` | one-request `Probe` with no body field; `SignerName()` | ✓ EXISTS + SUBSTANTIVE | 74 lines; `SignerName()` (`graphql.go:65`) returns `none` under the noop signer |

**Artifacts:** 26/26 verified (the previous pass's 25, plus `internal/audit/source.go`; `internal/tui/model.go` and `readonly_test.go` stand as they did, and `internal/xweb/volatile/registry.json` — the new check's evidence — is now in the covered-file set)

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/audit` | the tree | file reads only | ✓ WIRED | No import of `internal/store`, `internal/xweb`, `internal/cli`; asserted by test. The consequence is visible in `leak.go`, which labels source-derived surfaces and hands the CLI-owned outputs to `internal/cli`'s sweep |
| `x-terminal audit` | `internal/audit` | `audit.Run(repoRoot())` via the `auditRun` seam | ✓ WIRED | Same checks as `go test ./internal/audit/...`; both green this pass (10 checks / 0 findings / 893 files over the real tree) |
| the source check | every Go scan | `parseGoSources` (`source.go:55`) | ✓ WIRED | All six Go-scanning checks call it; a file they cannot parse is named once by `source` and counted nowhere else — re-measured this pass (the readonly/transport/timers/persistence/account counts drop by one, the total drops by the same five) |
| the classification table | the registry snapshot | `checkOpNames` → `registryNames` (`readonly.go:161-200`, `:130-147`) | ✓ WIRED | From the fix round: the snapshot is `//go:embed`ed (`registry.go:53`), so a registry the build does not carry is a build failure rather than a silent empty set; an unreadable one is a finding, an empty one is a finding, and a stale entry is a finding naming the table's own file (three plants, fix-round table rows 1-3) |
| registry JSON | read-only check | data read, with the decode failure named | ✓ WIRED | `registryNames` returns the error and `checkReadonly` turns it into a finding (`:210-213`); re-run plant exit 1 |
| `doctor` mirror line | `store.ReadFacts` | `store.ReadFacts(cfg.SavedDBPath())` | ✓ WIRED | No `store.Open` on that path; mtime test re-run and passing |
| `--probe` | the choke point | `Client.Do` → `Limiter.Do` | ✓ WIRED | Counting transport proves one request |
| every failing doctor line | `RUNBOOK.md#<anchor>` | `RunbookAnchor(checkState(r))` + `runbookAnchors` | ✓ WIRED | Both directions asserted: the table is total over failing outcomes (12 + 7 + 3 = 22 rows) and every pointer resolves to a heading; the plants re-run (row deleted, pointer retargeted, heading renamed) |
| a covered screen | a stored frame | cross-product table → `readFrame` → `TestGoldenFrames` | ✓ WIRED | A deleted covered frame is a test failure, and a changed byte is one too (both re-run) |
| the saved pane's three state branches | the pane's grammar | `m.grammar()` → `CapabilityBlock`/`StateBlockWithMirror`/`SavedEmptyState` | ✓ WIRED | Reverting the call sites to `render.Plain` fails the shipped-path arm in 14 state×width subtests / 216 findings (re-run) |
| the covered frames | the leak scan | render: `storedFrames` → `coveredLeakFindings` over every covered frame; tui: `coveredTUIFrames` → `tuileakFindings` over the 24 | ✓ WIRED | Both directions have in-package plant proofs whose base frames are a covered thread frame and a covered saved row frame (render) and a covered people frame (tui); the tui scan requires the 24-frame floor and the alias-presence arm |
| the frame enumerator | the covered check table | `frameKinds()` ↔ `frameKindChecks()` | ✓ WIRED | The join test requires one row per declared kind and no more; a sixth kind fails by count (re-run) |
| a runbook section | a built command | `findBuiltCommand` in the test | ✓ WIRED | Every section names one (14 lines); the matcher accepts a bare verb when `x-terminal` appears in the same line |
| the CLI's output surfaces | `audit.Sweep` | `realLeakSurfaces(t)` → `audit.Sweep("leak", surfaces)` | ✓ WIRED | One search function for both the audit's own check and the CLI's producers; the permanent planted arm requires it to fail on a revealed cookie (re-run) |
| the printed promise | the pinned destination set | `writeDestinations` → `persistencePromise()` → `persistenceDestinationNames` | ✓ WIRED | The names, the order and the printed text are asserted; renaming `config` fails the pin by name (re-run) |
| the store's own file failures | the "mirror unreadable" state | `store.FileError` → `errors.As` in `internal/cli/feed.go:129-130` | ✓ WIRED | The fix round's mark; the store suite's `TestTheFilesOwnReadFailuresCarryTheMark` passes and the audit's `store-source` surface reads the same package (0 findings). The whitelist's behaviour is phases 06/07's truth, not this phase's || the store's own file failures | the "mirror unreadable" state | `store.FileError` → `errors.As` in `internal/cli/feed.go:129-130` | ✓ WIRED | This round's mark; the store suite's `TestTheFilesOwnReadFailuresCarryTheMark` passes and the audit's `store-source` surface reads the same package (0 findings). The whitelist's behaviour is phases 06/07's truth, not this phase's |
| the keymap table (`internal/keys/keys.go`) | the dispatcher (`internal/tui/update.go`) | `keys.Rows()` ↔ `handledActions`, joined by `internal/tui/dispatch_test.go` | ✓ WIRED (partially — see anti-pattern 11) | This round's join: `handledActions` (30 actions) is set-equal to the table's 30 rows and to `press`'s 30 arms, and both directions fail on the round's own plants (guard table rows 1-2). Its third arm reads case labels anywhere in the package rather than inside `press`, so an arm deleted from `press` while its action stays a case label elsewhere is invisible to it (guard table row 5) |

**Wiring:** 16/16 connections verified (the previous pass's 15, plus the keymap⇄dispatcher join)

## Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| SEC-03: No code path mutates the account — no post/reply/like/repost/follow/bookmark write call anywhere in the build | ✓ SATISFIED | No write operation name is spelled in non-test source (table + arm, re-run), no non-GET method can be constructed (allowlist + method arm, re-run), and the write scan runs over every non-test file (fifth-destination plant, re-run); the table's own entries are now checked against the registry (`opnames`), so a write name the build cannot send cannot sit in the table either |
| OPS-03: `doctor` shows presence facts only and never a token value | ✓ SATISFIED | Mirror/signer/media/budget/session lines are facts; the fixture-cookie and base64 assertions fail when a token is planted; the doctor's text and JSON are swept as real surfaces; the exit-code rule is asserted per stubbed outcome; the package's tests are hermetic (poisoned-HOME check, 68 names identical) |
| OPS-06: A runbook covers operation-id rotation, features-blob drift, signer break, session expiry, 429, soft ban and accidental distribution | ✓ SATISFIED | All seven cases plus the mirror, likes endpoint, load shed, endpoint retired, empty body and unclassified; each four-part with a built command; a state without a section, a row without a heading and a heading without a state each fail the build (all plants re-run this pass) |
| Roadmap SC1 (presence facts, never a token) | ✓ SATISFIED | As OPS-03 |
| Roadmap SC2 (following the runbook repairs each case without a code change) | ✓ SATISFIED | Every repair is a command or a file edit except the signer-break escape hatch, which the section says to stop and report and names `internal/xweb/graphql.go` for a source change — honest rather than pretended |
| Roadmap SC3 (a repo-wide audit proves no write call exists, and re-runs the cookie-leak sweep across `doctor` output, logs, error paths and crash output) | ✓ SATISFIED | The write half is proven with the destination set pinned by name; the sweep is real for seven surfaces produced by their own code, with a permanent planted arm; and the audit can no longer pass a file it cannot read — the source check names it and the scans count the drop. The check set is now 10, and the count is pinned by name |
| Roadmap SC4 (everything since Phase 3 re-checked under the cover) | ✓ SATISFIED | Cross-product corpus (263 render + 56 tui frames = 319) with identity/product/URL absences, byte-identical keys, gated actions, the covered saved states stored and scanned, the covered likes-hedged frames pinned, the four covered notices on screen with a totality arm, the covered profile naming one account, and the frame enumerator joined to the per-kind checks; the product-noun arm is falsifiable on the stored bytes (re-run). The residual from the previous report — the saved screen's absence from `TestTheWholeSurfaceUnderACover`'s table — stands as anti-pattern 2: the frame-kind join covers the saved surface's covered-ness, its status field and its leak words, and this round's fix made the unreadable-mirror state genuinely reachable, but the test's other properties (same keys cost the same requests, `m` returns to the plain frame, the double-toggles come back byte-identical) are still not asserted for the saved screen |
| Roadmap SC5 (guardrail audit passes across the tree: no write path, no timer, no second persistence file, no second account, no distribution artifact) | ✓ SATISFIED | Timer/account/distribution/no-write-call arms are real and fail on plants; "no second persistence file" holds with the write scan over every non-test file, six named destinations pinned by name, and no silent exemption (two residuals recorded as ℹ️ info: a path inside an already-anchored development file, and the one dead constant the amendment round left behind) |

**Coverage:** 8/8 requirements satisfied

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/tui/corpus_test.go` | 803-878 | The tui covered-frame scan covers all 24 covered frames, but its word list is the 8 saved-surface repair words + the corpus's account identities — it carries neither render's `productNouns` table (`following`, `for you`, `post(s)`, `timeline(s)`, `feed`, `attachment`, `bookmarks`, `likes`) nor the plain reader's facts labels (`followers:`/`following:`/`posts:`/`joined:`) | ⚠️ Warning (unchanged) | The one surface that can carry a plain facts row is the one no scanner reads: a plain row `followers: 1204  following: 312  posts: 8901` planted into the committed `profile_gitlog_80.golden` passes `TestCorpusAudit` AND `TestTheCoveredTUIFramesLeakNoPlainVocabulary` and fails only the generator byte comparison |
| `internal/tui/cover_e2e_test.go` | 494-560 | `screens()` still lists Phase 2's nine screens; the saved screen was not added | ℹ️ Info | The leak side is covered by `TestTheCoveredSavedPaneNamesNothingOfTheProduct` (8 states × 2 widths × 4 styles, `:1443`), `TestCoveredFramesLeakNoRepairWords`, and the frame-kind join's saved row; what is still not asserted for the saved screen is that test's other properties (same keys cost the same requests, `m` returns to the plain frame, the double-toggles come back byte-identical) |
| `internal/audit/guards.go` | 353-379 | `classifyWrite` anchors `registry` to the whole file while `spike-recording` is anchored to the recorder function | ℹ️ Info | Re-confirmed in the previous pass: `os.WriteFile("/tmp/evil-anchored.log", …)` planted inside `writeAtomically` in `internal/xweb/volatile/registry.go` → 0 findings, exit 0, the new call printed as a resolved registry write. Not re-run this pass (the anchor code did not move); the residual is a new path inside an already-anchored development file |
| `internal/render/alias.go` | 138 | The alias dictionary contains `mirror`, one of the words the covered-frame leak scans treat as a leak | ℹ️ Info | Latent false positive, narrowed by the amendment round: the tui scanner masks each frame's own style aliases before scanning, which is exactly the case that would misfire (the CI alias `mirror-merge-65` in the 6 covered `search_people_*` frames); the render corpus keeps the six `search_people_*` rows outside the saved scans. Not live in either corpus |
| `internal/cli/runbook_test.go` | 191, 207 | `rule := "A repair is one request; …"` declared and then only discarded (`_ = rule`) | ℹ️ Info | Dead by the strictest reading; reads as the plan's rule kept as prose for the reader. `go vet` does not object; unchanged |
| `.planning/phases/08-operations-repair-path-and-audit/08-01-SUMMARY.md` | 87-88, 110 | The SUMMARY's provenance and its "Eight checks" paragraph still describe plan 01's own run (`8 checks, 0 findings, 782 files`) where the audit now runs 10 checks over 893 files; its planted row 4 also quotes the pre-amendment finding text (`resolves to a fifth destination`) where the check now says `resolves to no declared destination` | ℹ️ Info (widened this round) | Documentation drift in the plan's own record, not in the artefact: truth 9 is about the proofs existing and being recorded, and they are — the two later checks' plants are recorded in this report. The SUMMARY is history; this report does not claim it was updated |
| `internal/render/capture_tmux_test.go` | 160-166 | The per-run socket pin is enforced only by `TestTmuxSocketIsPrivateToThisRun`; a fixed-name plant is invisible to `-run TestCapture` | ℹ️ Info | The property is real and the harness cleans up (re-measured: 62 → 0 per screen, control 41), but a reader running the capture suite by prefix misses the arm |
| `internal/render/cover.go` + `internal/tui/testdata` | 195-210 | No test pins a cover stamp's `MonthYear` directly; the four `{joined}` clauses are held only by the 8 stored tui profile frames | ℹ️ Info | The clause renders (verified by reading the committed bytes) and removing one stamp's `MonthYear` fails that style's two golden frames (re-run), so the byte comparison is the pin; nothing asserts the *rule* "every cover stamp sets MonthYear" against a grammar added with the clause and without the stamp |
| `internal/render/copy.go` | 203 | `CopyNoticeMediaOne` is declared and never used — its only other appearance is the comment at `:200` that claims a use | ℹ️ Info | The amendment round deleted the nine dead constants it named (all ten enumerated names are gone from `internal/` and `cmd/`) and left this one; harmless, but the deletion was one short |
| `internal/audit/readonly.go` | 294-302 | request-builder allowlist wider than the plan named | ℹ️ Info | Adjudicated in (c) below and unchanged: the printed promise is a mutation claim and the method arm applies to every allowlisted package |
| `internal/tui/dispatch_test.go` + `internal/tui/update.go` | 105-162 (sweep), 227-329 (`press`) | The guard's stated property is "an arm no row can reach" caught in both directions, but `shellActionCases` (`dispatch_test.go:223-250`) accepts a case label **anywhere in the package**, while its `handledActions`-vs-`press` check does not read `press` at all: an arm deleted from `press` whose action is still a case label in `move` (`:381-397`), `coverBreaking` (`:342-346`) or `savedAction` (`saved.go:248-250`) leaves both directions green | ⚠️ Warning (new) | Deleting `press`'s arm for `ActionHalfDown`, `ActionHalfUp` or `ActionPageUp` passes the guard **and the entire 22-package suite** (0 failures) — the key becomes a silent no-op and nothing in the tree says so. `ActionPageDown` is caught only by one golden frame. Consequential arms (media/copy-link/open-post/bookmarks/likes/quit/panic) are caught by behavioural tests, not by the guard. The guard is not the exhaustiveness proof its comment advertises; adjudication (g) records the mechanism and what the round would need to close it |

**Anti-patterns:** 11 found (0 blockers, 2 warnings, 9 info)

## Adjudication

**(a) The covered-frame product-noun arm being fixture-only (`coveredSafeList`/`scrubCoverBody`).** Not a gap; the advisory is kept in frontmatter, and this pass re-ran both halves of its evidence: a same-width substitution in a committed covered frame fails it with the frame named (`corpus_test.go:390`), and the same `productNouns` table is armed by a permanent in-process arm in `internal/render/status_test.go`. The tui half has its own scanner with a floor, alias masking and a plant proof, so the advisory is narrower than it was; the hole that remains is anti-pattern 1.

**(b) The bare-Ghostty panic leg.** Deferred with reason, recorded in frontmatter and in Deferred below. The multiplexer leg — the mechanism the roadmap's DSGT-02 note calls out as the hard one — was re-measured on this host this pass (62 labelled lines → 0 on every later screen, control leaking 41). Deferring is not the same as marking the truth unverified: the leg is code that exists, is wired, and skips by design; the missing input is a window, not an implementation.

**(c) `readonly.go`'s documented wider request-builder allowlist.** It does not weaken the promise's wording; unchanged from the previous pass's adjudication, and this pass's plant put all three transport arms on one call site (the strongest form of the argument).

**(d) The new check's honesty as a check.** `opnames` proves a subset, not an equivalence: a registry operation the table does not classify is not its business (that is `readonly`'s, for a name spelled in source, and the write rule's, for the request the build can make). Its three failure modes are all falsified on the real tree, and its vacuity arms are in-package. The one thing it cannot see — a name assembled at run time from the registry — is exactly why the transport check exists beside it, which the table's own doc comment says.

**(e) The fix round's claims against the tree.** Verified claim by claim in the first table: the tenth check, its pin, its vacuity arms, the stale entry's removal, the store's `FileError` mark (cross-checked, not re-planted — another phase's truth), the saved frame's reachable unreadable state, and the three documented promises. Nothing in the round moved a golden byte or a must-have of this phase.

**(f) The previous pass's recorded warnings against the new tree.**

**(g) The guard round's claims against the tree — and the guard's own limit.** The round's three claims hold: one arm per action, an enumerable set that equals the arms both ways, and an exported copy (row 1's mechanical measurement), with the compiler refusing a bogus case label (row 3) and the two plants the author reported both failing (rows 1-2). The claim it does **not** earn is exhaustiveness over the dispatcher: `TestTheShellDeclaresOnlyActionsTheKeymapDeclares`'s last arm (`dispatch_test.go:151-161`) asks whether a declared action is a case label *anywhere in the shell*, and `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` compares `handledActions` to `keys.Rows()`, never to `press`'s arms. Since every movement action is also a case label in `move`, the three cover numbers in `coverBreaking`, and `bookmarks`/`likes` in `savedAction`, an arm dropped from the main switch is invisible to both (row 5): `ActionHalfDown`, `ActionHalfUp` and `ActionPageUp` can be deleted from `press` with the guard **and all 22 packages** green, so the key silently stops working. *Adjudication:* this is a defect in the guard's **claim**, not in the phase's deliverable — nothing this phase's must-haves rest on the guard, `press`'s arms are correct on the frozen tree, and the flows are covered by behavioural tests — so it is recorded as anti-pattern 11 (⚠️ Warning), not a gap. The honest fix (not made here, read-only) is to read `press`'s own arms, not the package's case labels: the sweep could require each action to be a case label inside `press`'s body, or the enumeration could be generated from the arms (as `keys`' subcommand already generates its output from `Rows()`) rather than hand-kept beside them.

## Deferred (environment-gated, not blocking)

### 1. The bare-Ghostty panic leg
**Run:** in a bare Ghostty window (TMUX unset, `TERM_PROGRAM=ghostty`, 80 columns, repository root) with a post-Phase-3 covered screen on screen: `go test ./internal/render -run 'TestCapturePanic|TestCaptureInGhostty' -count=1 -v`
**Expect:** after panic, six pages of Ghostty scrollback hold only the panic frame's own text; the run prints which screens it drove.
**Why deferred:** no Ghostty window exists in this environment and the leg skips with a printed reason (`capture_panic_test.go:290`). The multiplexer leg was measured here.

### 2. The live `doctor` run against a real session
**Run:** `x-terminal doctor` and `x-terminal doctor --probe Viewer` on the machine that holds the real session.
**Expect:** the registry, mirror, signer, session, check and request lines print; any failing line names a `RUNBOOK.md` anchor; `--probe` spends one request and prints one state word plus remaining/reset.
**Why deferred:** this host has a live account, and running the verb would spend real requests against the account's budget, which the phase's own request discipline forbids for verification. The doctor's lines, exit codes and report shapes are asserted hermetically and the report builders were read.

## Verification Metadata

**Verification approach:** re-verification of the frozen tree — goal-backward from ROADMAP.md's Phase 8 goal and success criteria plus the three plans' `must_haves`, with the guard round's claims re-falsified one by one (its two plants, its compiler floor, and the arm-deletion probe that found its blind spot), the previous pass's fix-round claims re-planted and the amendment round's twelve re-run in kind, and every truth re-adjudicated. Plants ran in `/tmp/v8finaltree` and `/tmp/v8t3` (byte copies of the repository, restored by `rsync --delete` from a pristine copy after each plant; the repository itself was never edited — verified by a whole-tree manifest). Some plants used the CLI (`go run ./cmd/x-terminal audit`) and the rest the audit's, the corpus's and the guard's own test entry points, because a tree plant that breaks the build (an unparseable `internal/xsource/home.go`, a removed `internal/render/copy.go`) must be read through a package that still parses.
**Must-haves source:** PLAN frontmatter (`08-01-PLAN.md`, `08-02-PLAN.md`, `08-03-PLAN.md`) merged with ROADMAP.md Phase 8 success criteria and REQUIREMENTS.md SEC-03/OPS-03/OPS-06.
**Automated checks:** `go build ./...` exit 0; `go vet ./...` exit 0; `gofmt -l internal cmd` empty; `go test ./... -count=1` exit 0 — **1573 `--- PASS` lines (694 top-level tests + 879 subtests/nested), 0 failed**, 8 skipped by design (7 roots + `TestCapturePanic/the_window`), 22 packages; the previous pass's 1571 plus exactly the guard's two new top-level tests, so nothing the round touched removed an assertion. The phase's suites (`internal/audit`, `internal/cli`, `internal/render`, `internal/tui`, `internal/store`, `internal/xweb`, `internal/xweb/volatile`, `internal/domain`, `internal/session`, `internal/keys`) re-run once inside `unshare -rn` with `lo` up and `GOPROXY=off` — all `ok`, so nothing in this phase reaches the network (external TCP is unreachable in that namespace: `/dev/tcp/1.1.1.1/443` → `Network is unreachable`); `go run ./cmd/x-terminal audit` exit 0 with 10 checks / 0 findings / 893 files inspected, the same arithmetic as the previous pass (`source 99`, `readonly 94`, `opnames 1`, `transport/timers/persistence/account/boundaries 99`, `distribution/leak 102`). The `internal/cli` suite is hermetic under a poisoned HOME: with `HOME`/`XDG_CONFIG_HOME`/`XDG_DATA_HOME` at an empty dir and the module cache pinned, the same 68 top-level test names ran and passed.
**Tree untouched by the suite:** a whole-tree manifest (`mtime size path`, 542 files outside `.git` and `.planning` — the previous pass's 541 plus `internal/tui/dispatch_test.go`) is identical before and after the full suite, and both `testdata` trees have the same aggregate sha256 (`18f992e7d4dee5e8146f9827ae86fcce9f0d5d99ca953e0e79d608e0c621c636`) before and after — equal to the digest the previous report recorded, so no golden byte moved in this round.
**Falsification performed this pass (in the scratch copies, restored after each; the previous pass's 30 plants re-run, plus the guard round's own plants and the arm-deletion probe — 50-odd runs plus the permanent in-package arms):** (1) the stale `ListLatestTweetsTimeline` back in the table → `opnames` finding naming `internal/audit/readonly.go` ("the table classifies \"ListLatestTweetsTimeline\" as a read and the registry does not hold it"), 10 checks / 1 finding / 893 files, exit 1; (2) an emptied snapshot → 33 findings / 893 files, exit 1; (3) `{broken json` snapshot → the `opnames` finding ("the names it holds cannot be read, so the table is unchecked") and `readonly`'s own decode finding, 2 findings, exit 1; (4) `opnames` removed from `Checks()` → `audit_test.go:123` fails by name ("the audit runs 9 checks … want the pinned 10") and the audit prints `9 checks, 0 findings, 892 files`, exit 0; (5) unparseable `internal/xsource/home.go` → `source: 99 files, 1 findings` + `internal/xsource/home.go:123: cannot parse: expected ')', found '{'`, with every Go scan dropping by one (readonly 94→93, transport/timers/persistence/account 99→98; boundaries stays 99), exit 1; (6) `source` removed and an unparseable `cmd/x-opsgen/main.go` carrying an `os.WriteFile` → `9 checks, 0 findings, 789 files inspected`, exit 0 (the pre-amendment silent pass); (7) `source` removed alone → the pin fails by name at `audit_test.go:123`; (8) `CreateTweet` spelled in `internal/xsource/home.go` → `internal/xsource/home.go:123: write operation name "CreateTweet" is spelled in source`, exit 1; (9) a new `internal/cli/zzplant.go` carrying a POST request, a ticker and an undeclared `os.WriteFile` → five findings at those lines, `10 checks, 5 findings, 902 files`, exit 1; (10) a root `Dockerfile` → `Dockerfile: distribution artifact "Dockerfile" exists`, `895 files`, exit 1; (11) a second `Config{SessionPath}` → `internal/config/config.go:555: second session-path resolution`, exit 1; (12) `config` renamed in the destination table → `guards_test.go:160` fails the set pin by name; (13) `internal/render/copy.go` removed → `no media-notice-constants surface`; (14) `internal/store/*.go` removed → `no store-source surface`; (15) a `throttle`→`timeline` substitution in `cover_ci_thread_80.golden` → `corpus_test.go:390` naming the frame (plus `TestGoldenFrames/cover_ci_thread_80`); (16) `cover_ci_thread_80.golden` deleted → `golden_test.go:352`; (17) `cover_gitlog_sync_notice_120.golden` deleted → `corpus_test.go:104` and `golden_test.go:352`; (18) a sixth frame kind → `cover_e2e_test.go:739`/`:743`; (19) `profileAccount` returning the opened handle under a cover → `cover_e2e_test.go:835`; (20) `profileAccount` naming nothing after the header → `:844`/`:851`; (21) `coverGitStamp.MonthYear` removed → tui `TestGoldenFrames/profile_gitlog_80` and `/120`; (22) the tmux socket fixed to `xterm-golden` → `capture_tmux_test.go:166`; (23) a user row planted into `search_80.golden` → tui `corpus_test.go:205`; (24) the covered likes hedge reverted → `saved_test.go:80` on all four styles plus 8 `cover_*_likes_hedged_{80,120}` frames; (25) the saved pane's call sites reverted to `render.Plain` → `cover_e2e_test.go:1467`/`:1476` (70 subtests across 14 state×width pairs / 216 findings); (26) a runbook section truncated to its Symptom → `runbook_test.go:148`/`:153`/`:169`/`:173`; (27) one `**Verify.**` replaced with prose → `:148`; (28) the `## Soft ban` heading renamed → `:82`/`:118`/`:203`/`:221`; (29) the doctor's `soft ban` row deleted → `doctor_test.go:486`/`:489`; (30) a doctor pointer retargeted → `runbook_test.go:118`. The guard round, this pass: (31) a 31st keymap row with a fresh action → `dispatch_test.go:63`/`:79`/`:128`/`:159`; (32) an action with an arm and an enumeration entry but no row → `dispatch_test.go:72`/`:79`/`:125`; (33) a bogus case label → `go build` fails `internal/tui/update.go:309:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction`; (34) `ActionSearchOrder`'s arm deleted from `press` → `dispatch_test.go:159`, caught; (35) the arm-deletion probe over all 30 `press` arms found the guard **blind** for exactly the 15 whose action is also a case label elsewhere in the package (`Next`/`Prev`/`HalfDown`/`HalfUp`/`PageDown`/`PageUp`/`First`/`Last` in `move`, `Media`/`CopyLink`/`OpenPost` in `coverBreaking`, `Quit`/`Panic` in the panic gate, `Bookmarks`/`Likes` in `savedAction`) and caught the other 15; `ActionHalfDown`/`ActionHalfUp`/`ActionPageUp` are invisible to the whole 22-package suite too and `ActionPageDown` is caught only by one golden frame. The permanent in-package plant arms (`TestCoveredLeakScanCatchesAPlant`, `TestTheCoveredTUIFrameLeakScanCatchesAPlant`, `TestCorpusAuditFailsOnPlantedViolation`, the planted half of `TestLeakSweepFindsNoCookieInRealOutput`) all pass.
**Human checks required:** 0 blocking. Two environment-gated measurements are recorded under Deferred (bare-Ghostty panic leg, live doctor run) — the tests exist and are wired; the environment does not.
**Not verifiable here:** the two deferred items above, and one thing this pass found rather than verified: the guard's blind spot (anti-pattern 11) is a property of the guard's construction, not of any input, so it cannot be "verified away" by a plant run — it was measured by deleting each arm in turn, and the measurement is what is recorded. The honest fix is a read-only change to `dispatch_test.go`, which this pass did not make. This verification ran without git commands, so `go.mod`/`go.sum` drift could not be diffed against the plan's baseline; the audit package's dependency-free claim is asserted by its own import test and consistent with `go list -deps` (one module package). The tmux-leg counts are harness-version-dependent (62/41 on this host's tmux) — the assertion is 0-after against a non-zero control, which this run confirms. A whole-suite plant run inside the scratch copy needs the repository's `.planning/research/x-ops-snapshot.json` present (two `internal/xweb`/`internal/xsource` tests read it), which the scratch copies carry. One plant was measured only in kind rather than repeated: the registry file-level anchor in `guards.go` (anti-pattern 3), whose code did not move this round.

---

*Verified: refreshed on the frozen tree after the guard round; every claim re-falsified, and the guard's own blind spot measured and recorded*
*Verifier: omp verifier subagent (adversarial, goal-backward, re-verification)*
