# AIW Skills Guide

  This directory contains a set of work skills for Codex/AIW. These are not isolated commands. They
  form a workflow toolkit that breaks work into stages: understanding the problem, shaping a solution,
  defining the spec, splitting tasks, implementing code, reviewing delivery, and keeping session and
  collaboration state in sync.

  This guide answers four questions:

  1. What problem does each skill solve?
  2. When should it be used?
  3. What does it usually produce?
  4. Which skills does it pair well with?

  ## 1. Start With the Three Skill Groups

  ### 1. Thinking and Design

  These skills help turn vague ideas into something concrete enough to discuss and decide on:

  `ask-matt`, `grilling`, `grill-me`, `grill-with-docs`, `prototype`, `research`, `office-hours-
  finance`, `business-review`, `metrics-review`, `eng-review-finance`, `domain-modeling`, `codebase-
  design`, `improve-codebase-
  architecture`, `autoplan-finance`.

  ### 2. OpenSpec and Implementation

  These skills turn conclusions into engineering artifacts and move the code forward:

  `to-spec`, `to-tickets`, `implement`, `tdd`, `code-review`, `release-review`, `publish-github-
  issue`.

  ### 3. Collaboration, Maintenance, and Learning

  These skills handle task routing, session continuity, Git state, environment setup, and knowledge
  transfer:

  `handoff`, `resume-ext`, `resolving-merge-conflicts`, `triage`, `wayfinder`, `setup-matt-pocock-
  skills`, `edit-article`, `teach`, `writing-great-skills`.

  ## 2. The Most Common End-to-End Path

  A typical engineering request can move through the following path:

  ```text
  vague idea
    -> ask-matt
    -> grilling / grill-with-docs
    -> office-hours-finance (if it is a finance/operations problem)
    -> business-review
    -> metrics-review
    -> eng-review-finance
    -> to-spec
    -> to-tickets
    -> implement / tdd
    -> code-review
    -> release-review
    -> publish-github-issue (only when explicitly needed)

  You do not need to use every skill every time. The route above is the full-coverage path. In
  practice, start with the stage that is currently missing the most.

  autoplan-finance is the orchestrator for finance scenarios: when you want a complete PLAN.md in one
  pass, use it directly. If you only need to solve a local issue, use the relevant review skill
  instead.

  ## 3. Skill-by-Skill Guide

  ### grill-me: Interactive pressure testing only

  Purpose: challenge a plan or design through an ongoing interview without creating documents or
  implementation artifacts.

  ### prototype: Quickly validate a design question

  Purpose: build a throwaway prototype to test a state model, logic, or UI direction.

  ### research: Conduct cited research

  Purpose: investigate a question using high-trust sources, distinguish facts from inference, and
  capture the result as Markdown.

  ### diagnosing-bugs: Diagnose failures and regressions

  Purpose: use an evidence-driven diagnosis loop to find the cause of broken, failing, throwing, or
  slow behavior.

  ### edit-article: Edit and improve an article

  Purpose: restructure an article, improve clarity, and tighten prose; return a revision or explicit
  proposed edits.

  ### ask-matt: You do not know what to use next

  Purpose: choose the right engineering skill or AIW/OpenSpec flow for the current situation.

  When to use it: the request is short, the context is messy, or you are not sure whether to clarify
  first, write a spec first, or implement directly.

  Typical result: a recommendation for the next step and the order of work. It is a router, not a
  replacement for the full work done by later skills.

  How it pairs: usually the entry point, then hand off to grill-with-docs, office-hours-finance, to-
  spec, or implement.

  ### grilling: Stress-test the idea one question at a time

  Purpose: expose assumptions, boundaries, and decision dependencies by asking one question at a time.

  When to use it: you want to check whether a plan is truly thought through, or you suspect key
  scenarios have been missed.

  Typical result: a set of pressure-tested decisions, boundaries, and open issues. It is interview-
  oriented and does not automatically produce a full document set.

  How it pairs: after that, you can use domain-modeling to solidify terminology and decisions, then
  to-spec to turn the result into an OpenSpec change.

  ### grill-with-docs: Stress-test while also producing documents

  Purpose: during the grilling process, also capture ADRs, glossaries, and other design documents.

  When to use it: the discussion is still early, and terminology, architecture choices, or key
  decisions need to be recorded.

  Typical result: a clearer design understanding, along with architecture decisions and domain
  vocabulary that are updated during the discussion.

  How it pairs: it usually comes before to-spec; if this is a finance request, it can also be followed
  by office-hours-finance or autoplan-finance.

  ### office-hours-finance: Clarify finance/operations problems

  Purpose: identify the real business problem, stakeholders, decision flow, scope, and unknowns.

  When to use it: someone asks for “a report, backend, risk dashboard, or operations tool” but has not
  explained who will use it, what decision it supports, or what success looks like.

  Typical result: a problem definition, users/roles, decision flow, scope reduction suggestions, and
  unknowns.

  How it pairs: usually before business-review; it clarifies the problem, but does not approve the
  project.

  ### business-review: Decide whether it is worth doing

  Purpose: decide APPROVE, REDUCE, or HOLD based on business value, cost, risk, and alternatives.

  When to use it: you need to decide whether a request, report, workflow, or platform capability is
  worth investing in.

  Typical result: a business decision with rationale, scope adjustments, and conditions for moving
  forward.

  How it pairs: usually follows office-hours-finance, and its outcome feeds into metrics-review or
  eng-review-finance.

  ### metrics-review: Define and review metrics

  Purpose: make metric names, formulas, sources, semantics, time windows, refresh frequency,
  ownership, and consistency risks explicit.

  When to use it: you are building KPIs, financial reports, management dashboards, or risk metrics, or
  you find that numbers do not match across systems.

  Typical result: a metric definition table, data-source mapping, semantic conflicts, and governance
  responsibility.

  How it pairs: usually used after business value is confirmed, then handed to eng-review-finance to
  design the data flow and implementation.

  ### eng-review-finance: Review the technical design

  Purpose: check system boundaries, data flow, module responsibilities, permissions, auditability,
  failure modes, observability, and testing strategy.

  When to use it: the finance or operations need is clear, and you need to confirm whether the
  technical approach is buildable, auditable, and recoverable.

  Typical result: technical design review comments, a risk list, data and permission design, and
  testing and monitoring requirements.

  How it pairs: usually follows metrics-review; close to release, use release-review.

  ### release-review: Review whether it is ready to ship

  Purpose: act as a release gate by checking schema/migration, data impact, metrics, permissions,
  audit, rollback, monitoring, and operational readiness.

  When to use it: implementation is mostly done and you are preparing to deploy or release, not when
  the requirement is just being defined.

  Typical result: release risks, blockers, pre-release checks, and a conclusion on whether the release
  is ready.

  How it pairs: it is a review, not a requirement-definition tool; it usually comes after eng-review-
  finance and code-review.

  ### autoplan-finance: Orchestrate a full finance plan

  Purpose: organize intake, business value, metrics, engineering design, and release preparation into
  a complete PLAN.md.

  When to use it: you are planning a finance backend, operations system, report, risk dashboard, or
  data project, and you want an end-to-end plan in one pass.

  Typical result: a planning document covering the problem, scope, decisions, metrics, architecture,
  risks, and release gates.

  How it pairs: it is an orchestrator, not a “write code immediately” tool. Once the plan is stable,
  use to-spec to convert it into OpenSpec, then use to-tickets and implement.

  ### domain-modeling: Build the domain model

  Purpose: unify terminology, identify concept relationships, record glossaries and architecture
  decisions, and actively test the model against boundary cases.

  When to use it: the same word means different things to different people or systems, or a design
  decision needs to become durable shared knowledge.

  Typical result: domain vocabulary, concept relationships, decision records, and clear boundary
  cases.

  How it pairs: often used with grilling, grill-with-docs, codebase-design, and to-spec.

  ### codebase-design: Improve module and code boundaries

  Purpose: use deep-module design language to analyze interfaces, responsibilities, encapsulation, and
  testability.

  When to use it: you do not know which module a feature belongs in, a module interface leaks too
  much, or the code is hard to test and hard for AI to navigate.

  Typical result: module boundary recommendations, interface design, responsibility adjustments, and
  directions for better testability.

  How it pairs: can be used before implement for design, or in code-review to explain structural
  issues.

  ### improve-codebase-architecture: Find opportunities to deepen the architecture

  Purpose: scan the codebase, generate a visual HTML report, then deep-dive into selected
  opportunities through follow-up questions.

  When to use it: you are dealing with a large or unfamiliar codebase and want to systematically find
  which modules are worth refactoring, rather than fixing only one local bug.

  Typical result: an architecture opportunity report and a deeper analysis of one selected
  opportunity.

  How it pairs: usually use it first to find direction, then use codebase-design, grilling, or to-spec
  to solidify the change.

  ### to-spec: Turn discussion into an OpenSpec change

  Purpose: combine the current conversation and understanding of the codebase to generate an OpenSpec
  change, including proposal, design, spec, and tasks artifacts.

  When to use it: the problem and key solution have already been discussed, and you no longer need
  more interviews. The next step is to turn them into an implementable specification.

  Typical result: a reviewable, implementable, and traceable OpenSpec change directory.

  How it pairs: usually after grill-with-docs, finance review, or architecture discussion; do not use
  it to “guess a complete solution from one vague sentence.”

  ### to-tickets: Break a solution into implementation slices

  Purpose: split a plan, spec, or current discussion into ordered tracer-bullet implementation items.

  When to use it: the solution already exists, but tasks.md is too coarse, or the dependencies and
  acceptance criteria between tasks are unclear.

  Typical result: OpenSpec tasks that are ordered, smaller in scope, and individually finishable.

  How it pairs: usually after to-spec and before implement.

  ### implement: Implement one selected task

  Purpose: complete one explicit implementation item based on an AIW Task and the corresponding
  OpenSpec change.

  When to use it: you already know which task to implement, and you should not rework the requirement
  or expand the scope here.

  Typical result: code changes, necessary tests or documentation updates, plus task status and
  verification records.

  How it pairs: follows to-spec / to-tickets; if test-first work is required, pair it with tdd.

  ### tdd: Implement with tests first

  Purpose: use the red-green-refactor loop to deliver one observable behavior as a small
  implementation cycle.

  When to use it: the user explicitly asks for TDD, test-first development, or the feature boundary is
  a good fit for behavior tests first.

  Typical result: failing tests first, then the minimal implementation, then code cleanup.

  How it pairs: usually the implementation method inside implement, not a replacement for OpenSpec.
  First define the task, then use TDD to complete it.

  ### code-review: Review code changes

  Purpose: compare changes against a given baseline, checking both repository standards and whether
  the work matches the original spec or request.

  When to use it: you need to review a branch, PR, or work after a specific commit, branch, tag, or
  merge-base.

  Typical result: bugs, regression risks, standards issues, and spec deviations, ordered by severity.

  How it pairs: usually after implement and before release-review. It is a code review, not a release
  review.

  ### publish-github-issue: Publish an OpenSpec change to GitHub Issue

  Purpose: publish an OpenSpec change as a managed GitHub Issue projection while keeping the local
  OpenSpec as the source of truth.

  When to use it: the user explicitly asks to publish a particular change to GitHub Issues.

  Typical result: a GitHub issue projection and linkage information back to the local change.

  How it pairs: it is not a normal “create issue” skill, and it does not replace to-spec. Use it only
  when the local spec is ready and external collaboration is explicitly needed.

  ### handoff: Prepare a handoff for the next session

  Purpose: compress the current conversation into a handoff document that the next agent can continue
  from.

  When to use it: the session is getting long, you need to switch threads, or you want to pass the
  current progress to another agent.

  Typical result: a summary of the current task, completed work, remaining work, risks, next steps,
  and key files.

  How it pairs: usually used before pausing implementation or switching sessions, and can be paired
  with resume-ext.

  ### resume-ext: Find and resume historical sessions

  Purpose: list local Codex sessions for the current workspace and help you choose and resume one of
  them.

  When to use it: you know the work happened in another session, but you do not want to restart or
  lose context.

  Typical result: a list of historical sessions and a resume command.

  How it pairs: handoff leaves the trail, and resume-ext restores the session. They solve opposite
  directions of the same continuity problem.

  ### resolving-merge-conflicts: Resolve Git merge conflicts

  Purpose: handle an existing merge or rebase conflict, understand both sides, preserve the correct
  behavior, and complete the conflict workflow.

  When to use it: Git is already in a conflicted state, with conflict files and an unfinished merge or
  rebase.

  Typical result: conflicts are resolved, the code is consistent, related checks are completed, and
  the merge or rebase can continue or finish.

  How it pairs: it is not a general code modification or review skill. Use it only when a Git conflict
  is actually present.

  ### triage: Triage issues and external PRs

  Purpose: process issues and PRs as a state machine by classifying them, validating them, asking
  necessary follow-up questions, and writing an agent-ready brief.

  When to use it: you receive many issues or external PRs and need to determine type, priority,
  executability, and next owner first.

  Typical result: labels, validation conclusions, clarifying questions, and a task brief that can be
  executed directly.

  How it pairs: after triage, you can move into grilling, to-spec, to-tickets, or implement.

  ### wayfinder: Plan large work across multiple sessions

  Purpose: break a task that exceeds one agent session into shared decision tickets and resolve
  dependencies one by one.

  When to use it: large migrations, cross-module refactors, long-running projects, or work that needs
  multiple agents to collaborate.

  Typical result: a decision map toward the goal, dependency order, and independently actionable work
  items.

  How it pairs: it sits upstream of large projects. Each decision or subtask can then use to-spec,
  implement, and handoff.

  ### setup-matt-pocock-skills: Configure the engineering workflow

  Purpose: set up AIW Task lifecycle, OpenSpec artifact management, triage labels, and domain
  documentation conventions.

  When to use it: you are enabling this engineering skill set in a project for the first time, or you
  need to repair or standardize project-level workflow configuration.

  Typical result: the project’s tasks, OpenSpec, labels, and domain documentation conventions are
  created or updated.

  How it pairs: usually only for project initialization or workflow migration, not as a daily skill
  for every request.

  ### teach: Learn a concept or skill

  Purpose: teach the user a concept or working method over a sustained, multi-session process.

  When to use it: you do not only want the agent to execute something; you also want to understand
  OpenSpec, metric governance, architecture design, or how a particular skill is used.

  Typical result: step-by-step explanation, exercises, feedback, and continuing learning context.

  How it pairs: it can be used to learn any workflow in this guide, and then you can return to the
  relevant skill to do the work.

  ### writing-great-skills: Write and improve skills

  Purpose: provide principles, structure, and predictability standards for writing skills.

  When to use it: you want to create a new skill, modify an existing one, or find that an agent
  frequently misuses a certain skill.

  Typical result: clearer triggers, steps, boundaries, output conventions, and validation methods.

  How it pairs: can be combined with teach and code-review; if you want to create or update a skill,
  read its guidance first.

  ## 4. Choose by Goal

   Your goal           First-choice skill               Next step
  ━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   You do not know     ask-matt                         Follow the recommendation into a specific
   where to start                                       flow
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Clarify a vague     grilling or grill-with-docs      to-spec or domain/finance review
   idea
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Plan a finance      autoplan-finance                 to-spec
   product or
   report
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Decide whether      business-review                  metrics-review or eng-review-finance
   it is worth
   doing
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Define KPI or       metrics-review                   eng-review-finance
   report semantics
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Design module       codebase-design                  to-spec or implement
   and architecture
   boundaries
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Scan a large        improve-codebase-architecture    Deepen the selected direction
   codebase for
   refactoring
   opportunities
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Write an            to-spec                          to-tickets
   OpenSpec change
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Split tasks.md      to-tickets                       implement
   into smaller
   tasks
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Start               implement                        code-review
   implementing one
   explicit task
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Test-first          tdd                              Pair it with implement
   development
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Review code         code-review                      Re-run after fixing issues, then do release
   changes                                              review
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Prepare for         release-review                   Release or roll back
   release
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Publish to          publish-github-issue             Only execute after explicit authorization
   GitHub Issue
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Handle merge/       resolving-merge-conflicts        Finish the Git workflow
   rebase conflicts
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Triage issues/      triage                           Enter the appropriate engineering flow
   PRs
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Plan a large        wayfinder                        Split and proceed item by item
   cross-session
   project
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Switch or resume    handoff / resume-ext             Continue the original task
   a session
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Initialize the      setup-matt-pocock-skills         Then begin the normal workflow
   project workflow
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Learn a concept     teach                            Then use the corresponding skill
  ──────────────────  ───────────────────────────────  ───────────────────────────────────────────────
   Write a new         writing-great-skills             Write, review, and try it out
   skill

  ## 5. Three Complete Examples

  ### Example A: A vague finance report request

  ask-matt
    -> grill-with-docs (terminology and key decisions are still unclear)
    -> office-hours-finance (clarify who reads it and what decision they make)
    -> business-review (APPROVE / REDUCE / HOLD)
    -> metrics-review (align formulas and data sources)
    -> eng-review-finance (data flow, permissions, audit)
    -> autoplan-finance or to-spec
    -> to-tickets
    -> implement
    -> code-review
    -> release-review

  ### Example B: An engineering change that has already been discussed clearly

  to-spec
    -> to-tickets
    -> implement
    -> tdd (if test-first is being used)
    -> code-review

  There is no need to run grilling again here. The job of to-spec is to synthesize the existing
  conclusions, not to interview the user again.

  ### Example C: A large refactor or long-running migration

  wayfinder
    -> triage / decision tickets
    -> codebase-design
    -> to-spec
    -> to-tickets
    -> implement
    -> handoff / resume-ext
    -> code-review

  If a merge or rebase conflict happens during implementation, temporarily switch to resolving-merge-
  conflicts, resolve it, then return to the original task.

  ## 6. Common Misunderstandings

  ### Is “review” used to define the content?

  Usually not. business-review, metrics-review, eng-review-finance, and release-review do produce
  definitions or recommendations, but their core job is to check, challenge, and gate a particular
  layer:

  office-hours-finance = clarify the problem first
  business-review      = review business value
  metrics-review       = review metric semantics
  eng-review-finance   = review the technical approach
  release-review       = review release readiness

  ### Are autoplan-finance and to-spec redundant?

  No. autoplan-finance is for “I want a complete decision and execution plan.” to-spec is for “these
  conclusions are clear enough now, please turn them into an OpenSpec change.” The first is planning
  orchestration, the second is spec delivery.

  ### Are grill-with-docs and autoplan-finance redundant?

  No. grill-with-docs digs into uncertainty and records ADRs/glossaries along the way; autoplan-
  finance consolidates multiple review stages in a finance workflow into a complete plan. The first is
  exploration-oriented, the second is orchestration-oriented.

  ### Should every skill be run every time?

  No. Skills are tools chosen by the gap they fill: if the requirement is clear, skip clarification;
  if it is not a finance problem, skip finance review; if there is no release plan, skip release-
  review. Use the full route only when the work crosses stages, carries high risk, or needs a complete
  record.

  ### Is publish-github-issue the same as creating a normal issue?

  No. It publishes an OpenSpec change as a managed GitHub Issue projection, and only when the user
  explicitly asks for external publication. Without that explicit request, keep maintaining the local
  OpenSpec.

  ## 7. Recommended Working Habits

  1. First decide whether the current gap is problem understanding, solution decision, spec,
     implementation, or review.

  2. Move only one clear stage at a time. Do not reopen unresolved requirements inside implement.
  3. Use to-spec to store stable requirements, to-tickets to store execution order, and implement to
     change code.

  4. Write review blockers back into the spec, design, or tasks instead of leaving them only in chat.
  5. For long tasks, use handoff before switching sessions and resume-ext when you return.
  6. Use publish-github-issue only when external collaboration is truly needed.

  ## 8. Quick Memory Aid

  Not clear      ask-matt / grilling
  Pressure-test  grill-me
  Validate design prototype
  Research       research
  Diagnose bugs  diagnosing-bugs
  Edit article   edit-article
  Need docs      grill-with-docs / domain-modeling
  Finance plan   autoplan-finance
  Write spec     to-spec
  Split tasks    to-tickets
  Write code     implement
  Test first     tdd
  Review code    code-review
  Ready to ship  release-review
  Post to GitHub publish-github-issue
  Conflict       resolving-merge-conflicts
  Big project    wayfinder
  Switch session handoff / resume-ext
  Learn things   teach
  Write skill    writing-great-skills
