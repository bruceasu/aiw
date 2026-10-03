const apiKey = process.env.OPENAI_API_KEY;
if (!apiKey) {
  console.error("OPENAI_API_KEY is not set.");
  process.exitCode = 2;
} else {
  const model = process.env.OPENAI_MODEL || "gpt-6-luna";
  try {
    const response = await fetch("https://api.openai.com/v1/responses", {
      method: "POST",
      headers: {
        authorization: `Bearer ${apiKey}`,
        "content-type": "application/json",
      },
      body: JSON.stringify({ model, input: "Reply with OK only.", max_output_tokens: 64 }),
      redirect: "error",
      signal: AbortSignal.timeout(30_000),
    });
    if (!response.ok) {
      console.error(`OpenAI API returned HTTP ${response.status}.`);
      process.exitCode = 1;
    } else {
      const result = await response.json();
      const output = typeof result.output_text === "string"
        ? result.output_text
        : (result.output || []).flatMap((item) => item.content || [])
          .filter((part) => part.type === "output_text")
          .map((part) => part.text || "").join("");
      console.log(`HTTP ${response.status}; model=${model}; text_received=${Boolean(output.trim())}`);
      console.log(`input_tokens=${result.usage?.input_tokens ?? "unknown"}; output_tokens=${result.usage?.output_tokens ?? "unknown"}`);
      if (!output.trim()) process.exitCode = 1;
    }
  } catch {
    console.error("OpenAI API request failed or timed out.");
    process.exitCode = 1;
  }
}
