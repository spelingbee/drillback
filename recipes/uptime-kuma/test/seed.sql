-- Minimal rows to make the data-sensitive checks meaningful. Column lists are
-- explicit so a schema change upstream fails loudly instead of silently seeding
-- nothing.
INSERT INTO user (username, password, active)
VALUES ('drilluser', '$2b$10$notarealhashnotarealhashnotarealhashnotarealhash', 1);

INSERT INTO monitor (name, type, url, interval, active, user_id)
VALUES ('drill monitor', 'http', 'http://kuma:3001/', 60, 0, 1);

INSERT INTO heartbeat (monitor_id, status, msg, time, ping, important, duration)
VALUES (1, 1, 'seeded by drillback recipe test', datetime('now'), 12, 1, 60);
