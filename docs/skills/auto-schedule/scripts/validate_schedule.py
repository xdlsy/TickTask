#!/usr/bin/env python3
"""Validate generated schedule.ics against task preferred time windows and recurrence days.

Usage:
  python3 validate_schedule.py --tasks config/todo.json --ics config/schedule.ics

Exit codes:
  0 - all tasks satisfy preferred time windows and recurrence constraints
  1 - validation errors found (mismatches)
  2 - file read/parse error
"""

import argparse
import json
import sys
from collections import defaultdict
from datetime import datetime, timedelta
from typing import Optional


def parse_ics_datetime(s: str) -> Optional[datetime]:
    """Parse an iCalendar DTSTART/DTEND value like '20260601T090000'."""
    s = s.strip()
    formats = ["%Y%m%dT%H%M%S", "%Y%m%dT%H%M%SZ"]
    for fmt in formats:
        try:
            return datetime.strptime(s, fmt)
        except ValueError:
            continue
    return None


def parse_ics(ics_path: str) -> list[dict]:
    """Parse .ics file and return list of VEVENT dicts."""
    events = []
    current = None
    with open(ics_path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line == "BEGIN:VEVENT":
                current = {}
            elif line == "END:VEVENT":
                if current:
                    events.append(current)
                current = None
            elif current is not None and ":" in line:
                key, _, value = line.partition(":")
                current[key] = value
    return events


def load_tasks(tasks_path: str) -> list[dict]:
    """Load tasks from todo.json."""
    with open(tasks_path, "r", encoding="utf-8") as f:
        data = json.load(f)
    return data.get("tasks", [])


def iso_weekday(dt: datetime) -> int:
    """Return ISO weekday: 1=Monday .. 7=Sunday."""
    return dt.isoweekday()


def validate(tasks: list[dict], events: list[dict]) -> tuple[list[dict], list[dict], list[dict]]:
    """Validate events against task preferred time windows and recurrence days.

    Returns (passed, failed, recurrence_mismatches) lists of validation results.
    """
    passed = []
    failed = []
    recurrence_mismatches = []

    task_map = {t["title"]: t for t in tasks}

    # Group events by task title
    events_by_task: dict[str, list[dict]] = defaultdict(list)
    for ev in events:
        summary = ev.get("SUMMARY", "")
        dtstart_raw = ev.get("DTSTART", "")
        if not summary or not dtstart_raw:
            continue
        break_keywords = ["休息", "午餐", "缓冲", "弹性", "break", "lunch"]
        if any(kw in summary for kw in break_keywords):
            continue
        events_by_task[summary].append(ev)

    # Collect all dates that have events
    all_dates: set[str] = set()
    for ev in events:
        dtstart_raw = ev.get("DTSTART", "")
        if dtstart_raw:
            dt = parse_ics_datetime(dtstart_raw)
            if dt:
                all_dates.add(dt.strftime("%Y-%m-%d"))

    all_dates_sorted = sorted(all_dates)

    # ---- Pass 1: Preferred time validation (existing) ----
    for ev in events:
        summary = ev.get("SUMMARY", "")
        dtstart_raw = ev.get("DTSTART", "")

        if not summary or not dtstart_raw:
            continue

        break_keywords = ["休息", "午餐", "缓冲", "弹性", "break", "lunch"]
        if any(kw in summary for kw in break_keywords):
            continue

        task = task_map.get(summary)
        if not task:
            continue

        pref_start = task.get("preferred_start_time", "")
        pref_end = task.get("preferred_end_time", "")
        if not pref_start or not pref_end:
            continue

        dt = parse_ics_datetime(dtstart_raw)
        if dt is None:
            continue

        actual_time = dt.strftime("%H:%M")
        date_str = dt.strftime("%Y-%m-%d")

        result = {
            "date": date_str,
            "task": summary,
            "preferred_start": pref_start,
            "preferred_end": pref_end,
            "actual_start": actual_time,
        }

        if pref_start <= actual_time <= pref_end:
            result["status"] = "ok"
            passed.append(result)
        else:
            result["status"] = "mismatch"
            result["reason"] = f"实际 {actual_time} 不在偏好窗口 {pref_start}-{pref_end} 内"
            failed.append(result)

    # ---- Pass 2: Recurrence day validation ----
    for task_title, task in task_map.items():
        is_recurring = task.get("is_recurring", False)
        rec_pattern = task.get("recurrence_pattern", "")
        rec_day = task.get("recurrence_day", 0)

        if not is_recurring or not rec_pattern:
            # Non-recurring tasks should appear at most once
            task_events = events_by_task.get(task_title, [])
            if len(task_events) > 1:
                dates = []
                for ev in task_events:
                    dt = parse_ics_datetime(ev.get("DTSTART", ""))
                    if dt:
                        dates.append(dt.strftime("%Y-%m-%d"))
                recurrence_mismatches.append({
                    "task": task_title,
                    "reason": f"非重复任务出现了 {len(task_events)} 次: {', '.join(sorted(dates))}，预期最多 1 次",
                    "expected": "最多 1 次",
                    "actual": f"{len(task_events)} 次",
                })
            continue

        task_events = events_by_task.get(task_title, [])
        if not task_events:
            continue

        if rec_pattern == "weekly":
            # Weekly tasks must appear exactly once on the correct day of week
            expected_weekday = rec_day  # 1=Mon .. 7=Sun
            day_names = ["", "周一", "周二", "周三", "周四", "周五", "周六", "周日"]

            if len(task_events) == 0:
                # Determine which date in the range matches the expected weekday
                if all_dates_sorted:
                    for date_str in all_dates_sorted:
                        dt = datetime.strptime(date_str, "%Y-%m-%d")
                        if iso_weekday(dt) == expected_weekday:
                            recurrence_mismatches.append({
                                "date": date_str,
                                "task": task_title,
                                "reason": f"每周重复任务应在{day_names[expected_weekday]}出现，但完全缺失",
                                "expected": f"应在 {day_names[expected_weekday]} 出现 1 次",
                                "actual": "缺失",
                            })
                            break
                    else:
                        recurrence_mismatches.append({
                            "date": "-",
                            "task": task_title,
                            "reason": f"每周重复任务({day_names[expected_weekday]})在日期范围内找不到对应日期",
                            "expected": f"应在 {day_names[expected_weekday]} 出现 1 次",
                            "actual": "缺失",
                        })
            elif len(task_events) > 1:
                dates = []
                for ev in task_events:
                    dt = parse_ics_datetime(ev.get("DTSTART", ""))
                    if dt:
                        dates.append(dt.strftime("%Y-%m-%d"))
                recurrence_mismatches.append({
                    "date": ", ".join(sorted(dates)),
                    "task": task_title,
                    "reason": f"每周重复任务应只出现 1 次，实际出现 {len(task_events)} 次",
                    "expected": f"应在 {day_names[expected_weekday]} 出现 1 次",
                    "actual": f"{len(task_events)} 次",
                })
            else:
                # Exactly 1 event - check if it's on the correct day
                for ev in task_events:
                    dt = parse_ics_datetime(ev.get("DTSTART", ""))
                    if dt is None:
                        continue
                    actual_weekday = iso_weekday(dt)
                    if actual_weekday != expected_weekday:
                        recurrence_mismatches.append({
                            "date": dt.strftime("%Y-%m-%d"),
                            "task": task_title,
                            "reason": f"每周重复任务应在{day_names[expected_weekday]}，实际排在{day_names[actual_weekday]}",
                            "expected": f"recurrence_day={expected_weekday} ({day_names[expected_weekday]})",
                            "actual": f"recurrence_day={actual_weekday} ({day_names[actual_weekday]})",
                        })

        elif rec_pattern == "daily":
            # Daily tasks should appear on EVERY day in the date range
            # (matches Go: matchesRecurrence returns true for all days)
            task_events = events_by_task.get(task_title, [])
            event_dates: set[str] = set()
            for ev in task_events:
                dt = parse_ics_datetime(ev.get("DTSTART", ""))
                if dt:
                    event_dates.add(dt.strftime("%Y-%m-%d"))

            # Determine expected dates from the full event date range
            if all_dates_sorted:
                start_date = datetime.strptime(all_dates_sorted[0], "%Y-%m-%d")
                end_date = datetime.strptime(all_dates_sorted[-1], "%Y-%m-%d")
                expected_dates: set[str] = set()
                d = start_date
                while d <= end_date:
                    expected_dates.add(d.strftime("%Y-%m-%d"))
                    d += timedelta(days=1)

                missing_dates = expected_dates - event_dates
                extra_dates = event_dates - expected_dates

                for date_str in sorted(missing_dates):
                    recurrence_mismatches.append({
                        "date": date_str,
                        "task": task_title,
                        "reason": f"每日重复任务缺少 {date_str} 的实例",
                        "expected": f"应在 {date_str} 出现",
                        "actual": "缺失",
                    })
                for date_str in sorted(extra_dates):
                    recurrence_mismatches.append({
                        "date": date_str,
                        "task": task_title,
                        "reason": f"每日重复任务在 {date_str} 多余出现",
                        "expected": f"不应在 {date_str} 出现",
                        "actual": "多余实例",
                    })

        elif rec_pattern == "monthly":
            # Monthly tasks must appear on the correct day of month
            for ev in task_events:
                dt = parse_ics_datetime(ev.get("DTSTART", ""))
                if dt is None:
                    continue
                actual_dom = dt.day
                if actual_dom != rec_day:
                    recurrence_mismatches.append({
                        "date": dt.strftime("%Y-%m-%d"),
                        "task": task_title,
                        "reason": f"每月重复任务应在第 {rec_day} 天，实际排在第 {actual_dom} 天",
                        "expected": f"每月第 {rec_day} 天",
                        "actual": f"每月第 {actual_dom} 天",
                    })

    return passed, failed, recurrence_mismatches


def print_report(passed: list[dict], failed: list[dict], recurrence_mismatches: list[dict]) -> str:
    """Generate a markdown validation report."""
    lines = []
    lines.append("## 偏好时段校验报告")
    lines.append("")
    lines.append("| 日期 | 任务 | 偏好时段 | 实际时段 | 状态 |")
    lines.append("|------|------|----------|----------|------|")

    for r in sorted(passed + failed, key=lambda x: (x["date"], x["task"])):
        pref = f"{r['preferred_start']}-{r['preferred_end']}"
        mark = "✅" if r["status"] == "ok" else "⚠️"
        lines.append(f"| {r['date']} | {r['task']} | {pref} | {r['actual_start']} | {mark} |")

    lines.append("")
    lines.append(f"**偏好时段: {len(passed)} 通过, {len(failed)} 不匹配**")

    if failed:
        lines.append("")
        lines.append("### ⚠️ 偏好时段不匹配的任务：")
        for r in failed:
            lines.append(f"- **{r['date']}** {r['task']}: {r['reason']}")

    if recurrence_mismatches:
        lines.append("")
        lines.append("## 重复日校验报告")
        lines.append("")
        lines.append(f"**重复日: {len(recurrence_mismatches)} 个不匹配**")
        lines.append("")
        lines.append("| 日期 | 任务 | 预期 | 实际 |")
        lines.append("|------|------|------|------|")
        for r in recurrence_mismatches:
            date_str = r.get("date", "-")
            lines.append(f"| {date_str} | {r['task']} | {r['expected']} | {r['actual']} |")
        lines.append("")
        lines.append("### ⚠️ 重复日不匹配详情：")
        for r in recurrence_mismatches:
            lines.append(f"- **{r.get('date', '-')}** {r['task']}: {r['reason']}")

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Validate schedule.ics against task preferences")
    parser.add_argument("--tasks", default="config/todo.json", help="Path to todo.json")
    parser.add_argument("--ics", default="config/schedule.ics", help="Path to schedule.ics")
    parser.add_argument("--json", action="store_true", help="Output JSON instead of markdown")
    args = parser.parse_args()

    try:
        tasks = load_tasks(args.tasks)
        events = parse_ics(args.ics)
    except Exception as e:
        print(f"ERROR: Failed to read input files: {e}", file=sys.stderr)
        sys.exit(2)

    passed, failed, recurrence_mismatches = validate(tasks, events)

    if args.json:
        print(json.dumps({
            "passed": passed,
            "failed": failed,
            "recurrence_mismatches": recurrence_mismatches,
        }, ensure_ascii=False, indent=2))
    else:
        report = print_report(passed, failed, recurrence_mismatches)
        print(report)

    total_failures = len(failed) + len(recurrence_mismatches)
    if total_failures > 0:
        print(f"\n❌ {len(failed)} 个偏好不匹配, {len(recurrence_mismatches)} 个重复日不匹配，需要修正", file=sys.stderr)
        sys.exit(1)
    else:
        print("\n✅ 全部偏好时段 + 重复日校验通过", file=sys.stderr)
        sys.exit(0)


if __name__ == "__main__":
    main()
