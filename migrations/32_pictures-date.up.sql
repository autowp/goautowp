alter table pictures
    add column taken_year year default null,
    add column taken_month month default null,
    add column taken_day day default null;
