//! 计划与下次运行时间。纯函数,全部以本地「墙上时间」计算。

use chrono::{Datelike, Duration, Local, NaiveDate, NaiveDateTime, TimeZone, Weekday};
use serde::{Deserialize, Serialize};

/// 运行计划。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum Schedule {
    /// 每隔若干分钟
    Interval { minutes: u32 },
    /// 每天固定时刻
    Daily { hour: u32, minute: u32 },
    /// 每周某天固定时刻;`weekday` 0 = 周一 … 6 = 周日
    Weekly { weekday: u32, hour: u32, minute: u32 },
}

const WEEKDAYS: [&str; 7] = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];

impl Schedule {
    /// 严格晚于 `after` 的下一次运行时间。
    pub fn next_after(&self, after: NaiveDateTime) -> NaiveDateTime {
        match *self {
            Schedule::Interval { minutes } => after + Duration::minutes(minutes.max(1) as i64),
            Schedule::Daily { hour, minute } => {
                let today = at(after.date(), hour, minute);
                if today > after { today } else { today + Duration::days(1) }
            }
            Schedule::Weekly { weekday, hour, minute } => {
                let target = weekday_of(weekday);
                let delta = (target.num_days_from_monday() as i64 - after.weekday().num_days_from_monday() as i64).rem_euclid(7);
                let candidate = at(after.date() + Duration::days(delta), hour, minute);
                if candidate > after { candidate } else { candidate + Duration::days(7) }
            }
        }
    }

    /// 人类可读的描述。
    pub fn describe(&self) -> String {
        match *self {
            Schedule::Interval { minutes } if minutes % 60 == 0 && minutes >= 60 => format!("每 {} 小时", minutes / 60),
            Schedule::Interval { minutes } => format!("每 {minutes} 分钟"),
            Schedule::Daily { hour, minute } => format!("每天 {hour:02}:{minute:02}"),
            Schedule::Weekly { weekday, hour, minute } => {
                format!("每{} {hour:02}:{minute:02}", WEEKDAYS[(weekday as usize).min(6)])
            }
        }
    }

    /// 校验取值范围。
    pub fn is_valid(&self) -> bool {
        match *self {
            Schedule::Interval { minutes } => (1..=7 * 24 * 60).contains(&minutes),
            Schedule::Daily { hour, minute } => hour < 24 && minute < 60,
            Schedule::Weekly { weekday, hour, minute } => weekday < 7 && hour < 24 && minute < 60,
        }
    }
}

fn weekday_of(n: u32) -> Weekday {
    match n {
        0 => Weekday::Mon,
        1 => Weekday::Tue,
        2 => Weekday::Wed,
        3 => Weekday::Thu,
        4 => Weekday::Fri,
        5 => Weekday::Sat,
        _ => Weekday::Sun,
    }
}

fn at(date: NaiveDate, hour: u32, minute: u32) -> NaiveDateTime {
    date.and_hms_opt(hour.min(23), minute.min(59), 0).expect("时分已被限制在合法范围")
}

/// 构造本地墙上时间(测试与界面共用)。
pub fn naive(y: i32, m: u32, d: u32, h: u32, min: u32) -> NaiveDateTime {
    NaiveDate::from_ymd_opt(y, m, d).expect("合法日期").and_hms_opt(h, min, 0).expect("合法时间")
}

/// Unix 秒 → 本地墙上时间。
pub fn to_naive(ts: i64) -> NaiveDateTime {
    Local.timestamp_opt(ts, 0).single().map(|d| d.naive_local()).unwrap_or_default()
}

/// 本地墙上时间 → Unix 秒(夏令时间隙取较早的有效时刻)。
pub fn to_ts(t: NaiveDateTime) -> i64 {
    Local.from_local_datetime(&t).earliest().map(|d| d.timestamp()).unwrap_or_else(|| t.and_utc().timestamp())
}

/// 周历里某一天的一次触发。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DaySlot {
    /// 当天固定时刻
    At { hour: u32, minute: u32 },
    /// 全天按间隔重复(不逐次列出)
    Every { minutes: u32 },
}

impl Schedule {
    /// 在一周里第 `weekday` 天(0 = 周一)的触发;当天不触发返回 `None`。
    pub fn slot_on(&self, weekday: u32) -> Option<DaySlot> {
        match *self {
            Schedule::Interval { minutes } => Some(DaySlot::Every { minutes }),
            Schedule::Daily { hour, minute } => Some(DaySlot::At { hour, minute }),
            Schedule::Weekly { weekday: w, hour, minute } => (w == weekday).then_some(DaySlot::At { hour, minute }),
        }
    }
}

/// 包含 `ts` 的那一周的七天 `(月, 日)`,周一在前;第二项是 `ts` 在这一周里的序号(0 = 周一)。
pub fn week_of(ts: i64) -> ([(u32, u32); 7], u32) {
    let d = to_naive(ts).date();
    let ix = d.weekday().num_days_from_monday();
    let monday = d - Duration::days(ix as i64);
    let mut out = [(0, 0); 7];
    for (i, o) in out.iter_mut().enumerate() {
        let day = monday + Duration::days(i as i64);
        *o = (day.month(), day.day());
    }
    (out, ix)
}

/// 当前 Unix 秒。
pub fn now_ts() -> i64 {
    Local::now().timestamp()
}

/// 把 Unix 秒格式化成 `MM-DD HH:MM`。
pub fn format_ts(ts: i64) -> String {
    to_naive(ts).format("%m-%d %H:%M").to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn 每隔若干分钟() {
        let s = Schedule::Interval { minutes: 30 };
        assert_eq!(s.next_after(naive(2026, 10, 22, 8, 50)), naive(2026, 10, 22, 9, 20));
    }

    #[test]
    fn 每天固定时刻() {
        let s = Schedule::Daily { hour: 9, minute: 30 };
        assert_eq!(s.next_after(naive(2026, 10, 22, 8, 0)), naive(2026, 10, 22, 9, 30));
        assert_eq!(s.next_after(naive(2026, 10, 22, 9, 30)), naive(2026, 10, 23, 9, 30), "必须严格晚于");
        assert_eq!(s.next_after(naive(2026, 12, 31, 10, 0)), naive(2027, 1, 1, 9, 30), "跨年");
    }

    #[test]
    fn 每周固定时刻() {
        // 2026-10-22 是周四
        let fri = Schedule::Weekly { weekday: 4, hour: 18, minute: 0 };
        assert_eq!(fri.next_after(naive(2026, 10, 22, 12, 0)), naive(2026, 10, 23, 18, 0));
        let thu = Schedule::Weekly { weekday: 3, hour: 9, minute: 0 };
        assert_eq!(thu.next_after(naive(2026, 10, 22, 12, 0)), naive(2026, 10, 29, 9, 0), "今天已过,顺延一周");
        assert_eq!(thu.next_after(naive(2026, 10, 22, 8, 0)), naive(2026, 10, 22, 9, 0));
        let mon = Schedule::Weekly { weekday: 0, hour: 9, minute: 0 };
        assert_eq!(mon.next_after(naive(2026, 10, 22, 8, 0)), naive(2026, 10, 26, 9, 0));
    }

    #[test]
    fn 周历触发与周范围() {
        assert_eq!(Schedule::Daily { hour: 9, minute: 0 }.slot_on(3), Some(DaySlot::At { hour: 9, minute: 0 }));
        let w = Schedule::Weekly { weekday: 0, hour: 10, minute: 30 };
        assert_eq!(w.slot_on(0), Some(DaySlot::At { hour: 10, minute: 30 }));
        assert_eq!(w.slot_on(1), None);
        assert_eq!(Schedule::Interval { minutes: 60 }.slot_on(6), Some(DaySlot::Every { minutes: 60 }));
        // 2026-10-22 是周四:这一周是 10-19 ~ 10-25
        let (days, ix) = week_of(to_ts(naive(2026, 10, 22, 12, 0)));
        assert_eq!(ix, 3);
        assert_eq!(days[0], (10, 19));
        assert_eq!(days[6], (10, 25));
        // 跨月:2026-11-01 是周日,这一周从 10-26 开始
        let (days, ix) = week_of(to_ts(naive(2026, 11, 1, 8, 0)));
        assert_eq!((ix, days[0], days[6]), (6, (10, 26), (11, 1)));
    }

    #[test]
    fn 描述与校验() {
        assert_eq!(Schedule::Interval { minutes: 120 }.describe(), "每 2 小时");
        assert_eq!(Schedule::Interval { minutes: 45 }.describe(), "每 45 分钟");
        assert_eq!(Schedule::Daily { hour: 7, minute: 5 }.describe(), "每天 07:05");
        assert_eq!(Schedule::Weekly { weekday: 6, hour: 20, minute: 0 }.describe(), "每周日 20:00");
        assert!(!Schedule::Daily { hour: 24, minute: 0 }.is_valid());
        assert!(!Schedule::Interval { minutes: 0 }.is_valid());
        assert!(Schedule::Weekly { weekday: 6, hour: 23, minute: 59 }.is_valid());
    }

    #[test]
    fn 时间戳往返() {
        let t = naive(2026, 10, 22, 9, 5);
        assert_eq!(to_naive(to_ts(t)), t);
    }
}
