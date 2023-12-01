--daily active users
select
	event_time::timestamp::date as event_date,
	service,
	count(distinct(anonymous_id)) as daily_active_users
from events.events as e
where e.event not like '%user_subscribed%'
and   e.event not like '%user_unsubscribed%'
group by event_date, service
