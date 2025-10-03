-- Reset database script
-- This will drop and recreate all tables

DROP TABLE IF EXISTS "users" CASCADE;
DROP TABLE IF EXISTS "tasks" CASCADE;
DROP TABLE IF EXISTS "task_progresses" CASCADE;
DROP TABLE IF EXISTS "daily_tasks" CASCADE;
DROP TABLE IF EXISTS "monthly_tasks" CASCADE;
DROP TABLE IF EXISTS "orders" CASCADE;
DROP TABLE IF EXISTS "reminders" CASCADE;
DROP TABLE IF EXISTS "financial_settings" CASCADE;
DROP TABLE IF EXISTS "calculation_histories" CASCADE;
DROP TABLE IF EXISTS "report_queries" CASCADE;

-- The tables will be recreated by GORM auto-migration
