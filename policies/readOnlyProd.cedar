permit(
  principal == workload::"afariasjara-agent",
  action in [action::"db:Query", action::"s3:Get", action::"dynamo:GetItem"],
  resource in dataset::"prod"
) when {
  context.readOnly == true && context.env == "prod" && context.risk < 70
};

forbid(
  principal == workload::"afariasjara-agent",
  action in [action::"db:Delete", action::"s3:Put", action::"service:Restart"],
  resource in dataset::"prod"
);
