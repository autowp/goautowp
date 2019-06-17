alter table pictures
    add column taken_year year default null,
    add column taken_month tinyint default null,
    add column taken_day tinyint default null;
