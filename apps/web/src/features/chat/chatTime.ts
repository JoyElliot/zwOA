const dayMilliseconds = 86_400_000;
const weekdays = ['日', '一', '二', '三', '四', '五', '六'];

function calendarDay(date: Date) {
  // Compare local calendar dates without counting the hours in a daylight-saving day.
  return Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()) / dayMilliseconds;
}

export function formatChatTime(date: Date, now: Date) {
  if (!Number.isFinite(date.getTime()) || !Number.isFinite(now.getTime())) return '';
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  const time = `${hours}:${minutes}`;
  const daysAgo = calendarDay(now) - calendarDay(date);
  if (daysAgo <= 0) return time;
  if (daysAgo === 1) return `昨天 ${time}`;
  if (daysAgo === 2) return `前天 ${time}`;

  const weekday = weekdays[date.getDay()];
  const daysSinceMonday = (now.getDay() + 6) % 7;
  if (daysAgo <= daysSinceMonday) return `星期${weekday} ${time}`;
  if (daysAgo <= daysSinceMonday + 7) return `上周${weekday} ${time}`;

  const month = date.getMonth() + 1;
  const day = date.getDate();
  return date.getFullYear() === now.getFullYear()
    ? `${month}月${day}日 ${time}`
    : `${date.getFullYear()}-${month}-${day} ${time}`;
}
