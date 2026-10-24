// Afarias19 - Anti-Brick - Commit 640c057
@id("readOnlyProd")
permit(principal == workload::"afariasjara-agent", action in [action::"db:Query", action::"s3:GetObject", action::"dynamo:GetItem", action::"api:Read"], resource in dataset::"prod") when { context.readOnly == true && context.env == "prod" && context.risk < 70 && context.health == "ok" };
@id("forbidDestructiveProd")
forbid(principal == workload::"afariasjara-agent", action in [action::"db:Delete", action::"db:Update", action::"s3:PutObject", action::"service:Restart", action::"trade:Order"], resource in dataset::"prod") when { context.env == "prod" };
