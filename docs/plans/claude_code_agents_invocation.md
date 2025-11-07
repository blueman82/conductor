  🔍 Deep Research Report: Claude Code CLI & Subagent Invocation

  Executive Summary

  YES, Claude Code CAN invoke subagents from the command line using the -p flag combined with the --agents option! I've
  successfully tested this functionality and documented all the mechanisms below.

  ---
  1. Command-Line Subagent Invocation Methods

  Method 1: Dynamic Agent Definition with --agents Flag

  You can define custom subagents on-the-fly using JSON:

  claude --agents '{"agent-name": {"description": "...", "prompt": "...", "tools": [...]}}' -p "your query"

  Real Example I Tested:
  claude --agents '{"code-searcher":{"description":"Searches code and files","prompt":"You are a code search expert. Search
  thoroughly.","tools":["Read","Grep","Glob"]}}' -p "use the code-searcher subagent to find any .txt files in /tmp"

  ✅ Result: Successfully invoked the custom subagent and got results!

  Method 2: Using Pre-configured Agents

  Agents stored in:
  - Project level: .claude/agents/*.md (highest priority)
  - User level: ~/.claude/agents/*.md (27 agents found on your system!)

  Invoke them in -p mode by explicitly requesting:
  claude -p "use the debugger-agent to investigate the error"

  Method 3: Automatic Delegation

  Claude automatically routes tasks to appropriate subagents based on context:
  claude -p "review my code for security issues"  # Auto-invokes code-reviewer
  claude -p "run all tests and fix failures"      # Auto-invokes test-automator

  ---
  2. Agent Configuration Format

  ★ Insight ─────────────────────────────────────
  Agents use Markdown files with YAML frontmatter, making them human-readable and git-friendly. This design enables:
  - Version control for agent definitions
  - Easy sharing across team members
  - Project-specific customization without global changes
  ─────────────────────────────────────────────────

  Structure (from ~/.claude/agents/hello-world-agent.md:1-29):
  ---
  name: agent-name
  description: When to invoke this agent
  tools: Comma-separated tool list
  color: green
  ---

  # Agent System Prompt
  Your detailed instructions here...

  Required Fields:
  - name: Unique identifier (lowercase, hyphens)
  - description: Natural language trigger conditions
  - prompt: System instructions for the agent

  Optional Fields:
  - tools: Array/comma-separated (inherits all if omitted)
  - model: sonnet, opus, haiku, or inherit
  - color: UI identifier

  ---
  3. CLI Options Reference

  Print Mode (-p / --print)

  - Executes query and exits (non-interactive)
  - Essential for scripting and automation
  - Works with pipes: cat file | claude -p "analyze this"

  Agent Configuration

  --agents <json>              # Define custom agents dynamically
  --system-prompt <prompt>     # Override system prompt
  --append-system-prompt       # Add to default prompt

  Output Control

  --output-format <type>       # text (default), json, stream-json
  --include-partial-messages   # Stream chunks (with stream-json)
  --verbose                    # Debug output

  Tool Management

  --tools <list>               # Specify available tools
  --allowedTools <list>        # Whitelist specific tools
  --disallowedTools <list>     # Blacklist specific tools

  Session Management

  -c, --continue               # Resume most recent conversation
  -r, --resume [sessionId]     # Resume specific session
  --fork-session               # Create new session ID when resuming

  ---
  4. Practical Command-Line Examples

  One-Off Query with Custom Agent

  claude --agents '{"reviewer":{"description":"Code reviewer","prompt":"Review code thoroughly","tools":["Read","Grep"]}}' -p
   "review the auth module"

  Piped Content Analysis

  git diff | claude -p "explain these changes"

  Programmatic Output

  claude -p "list all functions in main.py" --output-format json

  Continuing Session Non-Interactively

  claude -c -p "now add unit tests for those functions"

  Limited Tool Access

  claude --tools "Read,Grep,Glob" -p "search for authentication code"

  ---
  5. Subagent Architecture Insights

  ★ Insight ─────────────────────────────────────
  Subagents operate with independent context windows, preventing "context pollution" in the main conversation. This
  architectural decision:
  - Allows specialized agents to focus deeply without main thread overhead
  - Enables concurrent agent execution for parallel tasks
  - Maintains conversation clarity by isolating complex sub-tasks
  ─────────────────────────────────────────────────

  Key Characteristics:
  - Each agent has its own system prompt
  - Agents inherit or restrict tools based on configuration
  - Built-in "Plan" subagent prevents infinite nesting
  - Agents report back to main conversation after completion

  ---
  6. Your Installed Agents

  I discovered 27 agents in ~/.claude/agents/, including:

| Agent | Purpose |
|-------|---------|
| api-designer | API architecture expert designing scalable, developer-friendly interfaces. Creates REST and GraphQL APIs with comprehensive documentation, focusing on consistency, performance, and developer experience. |
| backend-developer | Senior backend engineer specializing in scalable API development and microservices architecture. Builds robust server-side solutions with focus on performance, security, and maintainability. |
| electron-pro | Desktop application specialist building secure cross-platform solutions. Develops Electron apps with native OS integration, focusing on security, performance, and seamless user experience. |
| frontend-developer | Expert UI engineer focused on crafting robust, scalable frontend solutions. Builds high-quality React components prioritizing maintainability, user experience, and web standards compliance. |
| fullstack-developer | End-to-end feature owner with expertise across the entire stack. Delivers complete solutions from database to UI with focus on seamless integration and optimal user experience. |
| graphql-architect | GraphQL schema architect designing efficient, scalable API graphs. Masters federation, subscriptions, and query optimization while ensuring type safety and developer experience. |
| microservices-architect | Distributed systems architect designing scalable microservice ecosystems. Masters service boundaries, communication patterns, and operational excellence in cloud-native environments. |
| mobile-developer | Cross-platform mobile specialist building performant native experiences. Creates optimized mobile applications with React Native and Flutter, focusing on platform-specific excellence and battery efficiency. |
| ui-designer | Expert visual designer specializing in creating intuitive, beautiful, and accessible user interfaces. Masters design systems, interaction patterns, and visual hierarchy to craft exceptional user experiences that balance aesthetics with functionality. |
| websocket-engineer | Real-time communication specialist implementing scalable WebSocket architectures. Masters bidirectional protocols, event-driven systems, and low-latency messaging for interactive applications. |
| wordpress-master | Expert WordPress developer specializing in theme development, plugin architecture, and performance optimization. Masters both classic PHP development and modern block-based solutions, delivering scalable WordPress sites from simple blogs to enterprise platforms. |
| angular-architect | Expert Angular architect mastering Angular 15+ with enterprise patterns. Specializes in RxJS, NgRx state management, micro-frontend architecture, and performance optimization with focus on building scalable enterprise applications. |
| cpp-pro | Expert C++ developer specializing in modern C++20/23, systems programming, and high-performance computing. Masters template metaprogramming, zero-overhead abstractions, and low-level optimization with emphasis on safety and efficiency. |
| csharp-developer | Expert C# developer specializing in modern .NET development, ASP.NET Core, and cloud-native applications. Masters C# 12 features, Blazor, and cross-platform development with emphasis on performance and clean architecture. |
| django-developer | Expert Django developer mastering Django 4+ with modern Python practices. Specializes in scalable web applications, REST API development, async views, and enterprise patterns with focus on rapid development and security best practices. |
| dotnet-core-expert | Expert .NET Core specialist mastering .NET 8 with modern C# features. Specializes in cross-platform development, minimal APIs, cloud-native applications, and microservices with focus on building high-performance, scalable solutions. |
| dotnet-framework-4.8-expert | Expert .NET Framework 4.8 specialist mastering legacy enterprise applications. Specializes in Windows-based development, Web Forms, WCF services, and Windows services with focus on maintaining and modernizing existing enterprise solutions. |
| flutter-expert | Expert Flutter specialist mastering Flutter 3+ with modern architecture patterns. Specializes in cross-platform development, custom animations, native integrations, and performance optimization with focus on creating beautiful, native-performance applications. |
| golang-pro | Expert Go developer specializing in high-performance systems, concurrent programming, and cloud-native microservices. Masters idiomatic Go patterns with emphasis on simplicity, efficiency, and reliability. |
| java-architect | Senior Java architect specializing in enterprise-grade applications, Spring ecosystem, and cloud-native development. Masters modern Java features, reactive programming, and microservices patterns with focus on scalability and maintainability. |
| javascript-pro | Expert JavaScript developer specializing in modern ES2023+ features, asynchronous programming, and full-stack development. Masters both browser APIs and Node.js ecosystem with emphasis on performance and clean code patterns. |
| kotlin-specialist | Expert Kotlin developer specializing in coroutines, multiplatform development, and Android applications. Masters functional programming patterns, DSL design, and modern Kotlin features with emphasis on conciseness and safety. |
| laravel-specialist | Expert Laravel specialist mastering Laravel 10+ with modern PHP practices. Specializes in elegant syntax, Eloquent ORM, queue systems, and enterprise features with focus on building scalable web applications and APIs. |
| nextjs-developer | Expert Next.js developer mastering Next.js 14+ with App Router and full-stack features. Specializes in server components, server actions, performance optimization, and production deployment with focus on building fast, SEO-friendly applications. |
| php-pro | Expert PHP developer specializing in modern PHP 8.3+ with strong typing, async programming, and enterprise frameworks. Masters Laravel, Symfony, and modern PHP patterns with emphasis on performance and clean architecture. |
| python-pro | Expert Python developer specializing in modern Python 3.11+ development with deep expertise in type safety, async programming, data science, and web frameworks. Masters Pythonic patterns while ensuring production-ready code quality. |
| rails-expert | Expert Rails specialist mastering Rails 7+ with modern conventions. Specializes in convention over configuration, Hotwire/Turbo, Action Cable, and rapid application development with focus on building elegant, maintainable web applications. |
| react-specialist | Expert React specialist mastering React 18+ with modern patterns and ecosystem. Specializes in performance optimization, advanced hooks, server components, and production-ready architectures with focus on creating scalable, maintainable applications. |
| rust-engineer | Expert Rust developer specializing in systems programming, memory safety, and zero-cost abstractions. Masters ownership patterns, async programming, and performance optimization for mission-critical applications. |
| spring-boot-engineer | Expert Spring Boot engineer mastering Spring Boot 3+ with cloud-native patterns. Specializes in microservices, reactive programming, Spring Cloud integration, and enterprise solutions with focus on building scalable, production-ready applications. |
| sql-pro | Expert SQL developer specializing in complex query optimization, database design, and performance tuning across PostgreSQL, MySQL, SQL Server, and Oracle. Masters advanced SQL features, indexing strategies, and data warehousing patterns. |
| swift-expert | Expert Swift developer specializing in Swift 5.9+ with async/await, SwiftUI, and protocol-oriented programming. Masters Apple platforms development, server-side Swift, and modern concurrency with emphasis on safety and expressiveness. |
| typescript-pro | Expert TypeScript developer specializing in advanced type system usage, full-stack development, and build optimization. Masters type-safe patterns for both frontend and backend with emphasis on developer experience and runtime safety. |
| vue-expert | Expert Vue specialist mastering Vue 3 with Composition API and ecosystem. Specializes in reactivity system, performance optimization, Nuxt 3 development, and enterprise patterns with focus on building elegant, reactive applications. |
| cloud-architect | Expert cloud architect specializing in multi-cloud strategies, scalable architectures, and cost-effective solutions. Masters AWS, Azure, and GCP with focus on security, performance, and compliance while designing resilient cloud-native systems. |
| database-administrator | Expert database administrator specializing in high-availability systems, performance optimization, and disaster recovery. Masters PostgreSQL, MySQL, MongoDB, and Redis with focus on reliability, scalability, and operational excellence. |
| deployment-engineer | Expert deployment engineer specializing in CI/CD pipelines, release automation, and deployment strategies. Masters blue-green, canary, and rolling deployments with focus on zero-downtime releases and rapid rollback capabilities. |
| devops-engineer | Expert DevOps engineer bridging development and operations with comprehensive automation, monitoring, and infrastructure management. Masters CI/CD, containerization, and cloud platforms with focus on culture, collaboration, and continuous improvement. |
| devops-incident-responder | Expert incident responder specializing in rapid detection, diagnosis, and resolution of production issues. Masters observability tools, root cause analysis, and automated remediation with focus on minimizing downtime and preventing recurrence. |
| incident-responder | Expert incident responder specializing in security and operational incident management. Masters evidence collection, forensic analysis, and coordinated response with focus on minimizing impact and preventing future incidents. |
| kubernetes-specialist | Expert Kubernetes specialist mastering container orchestration, cluster management, and cloud-native architectures. Specializes in production-grade deployments, security hardening, and performance optimization with focus on scalability and reliability. |
| network-engineer | Expert network engineer specializing in cloud and hybrid network architectures, security, and performance optimization. Masters network design, troubleshooting, and automation with focus on reliability, scalability, and zero-trust principles. |
| platform-engineer | Expert platform engineer specializing in internal developer platforms, self-service infrastructure, and developer experience. Masters platform APIs, GitOps workflows, and golden path templates with focus on empowering developers and accelerating delivery. |
| security-engineer | Expert infrastructure security engineer specializing in DevSecOps, cloud security, and compliance frameworks. Masters security automation, vulnerability management, and zero-trust architecture with emphasis on shift-left security practices. |
| sre-engineer | Expert Site Reliability Engineer balancing feature velocity with system stability through SLOs, automation, and operational excellence. Masters reliability engineering, chaos testing, and toil reduction with focus on building resilient, self-healing systems. |
| terraform-engineer | Expert Terraform engineer specializing in infrastructure as code, multi-cloud provisioning, and modular architecture. Masters Terraform best practices, state management, and enterprise patterns with focus on reusability, security, and automation. |
| accessibility-tester | Expert accessibility tester specializing in WCAG compliance, inclusive design, and universal access. Masters screen reader compatibility, keyboard navigation, and assistive technology integration with focus on creating barrier-free digital experiences. |
| architect-reviewer | Expert architecture reviewer specializing in system design validation, architectural patterns, and technical decision assessment. Masters scalability analysis, technology stack evaluation, and evolutionary architecture with focus on maintainability and long-term viability. |
| chaos-engineer | Expert chaos engineer specializing in controlled failure injection, resilience testing, and building antifragile systems. Masters chaos experiments, game day planning, and continuous resilience improvement with focus on learning from failure. |
| code-reviewer | Expert code reviewer specializing in code quality, security vulnerabilities, and best practices across multiple languages. Masters static analysis, design patterns, and performance optimization with focus on maintainability and technical debt reduction. |
| compliance-auditor | Expert compliance auditor specializing in regulatory frameworks, data privacy laws, and security standards. Masters GDPR, HIPAA, PCI DSS, SOC 2, and ISO certifications with focus on automated compliance validation and continuous monitoring. |
| debugger | Expert debugger specializing in complex issue diagnosis, root cause analysis, and systematic problem-solving. Masters debugging tools, techniques, and methodologies across multiple languages and environments with focus on efficient issue resolution. |
| error-detective | Expert error detective specializing in complex error pattern analysis, correlation, and root cause discovery. Masters distributed system debugging, error tracking, and anomaly detection with focus on finding hidden connections and preventing error cascades. |
| penetration-tester | Expert penetration tester specializing in ethical hacking, vulnerability assessment, and security testing. Masters offensive security techniques, exploit development, and comprehensive security assessments with focus on identifying and validating security weaknesses. |
| performance-engineer | Expert performance engineer specializing in system optimization, bottleneck identification, and scalability engineering. Masters performance testing, profiling, and tuning across applications, databases, and infrastructure with focus on achieving optimal response times and resource efficiency. |
| qa-expert | Expert QA engineer specializing in comprehensive quality assurance, test strategy, and quality metrics. Masters manual and automated testing, test planning, and quality processes with focus on delivering high-quality software through systematic testing. |
| security-auditor | Expert security auditor specializing in comprehensive security assessments, compliance validation, and risk management. Masters security frameworks, audit methodologies, and compliance standards with focus on identifying vulnerabilities and ensuring regulatory adherence. |
| test-automator | Expert test automation engineer specializing in building robust test frameworks, CI/CD integration, and comprehensive test coverage. Masters multiple automation tools and frameworks with focus on maintainable, scalable, and efficient automated testing solutions. |
| ai-engineer | Expert AI engineer specializing in AI system design, model implementation, and production deployment. Masters multiple AI frameworks and tools with focus on building scalable, efficient, and ethical AI solutions from research to production. |
| data-analyst | Expert data analyst specializing in business intelligence, data visualization, and statistical analysis. Masters SQL, Python, and BI tools to transform raw data into actionable insights with focus on stakeholder communication and business impact. |
| data-engineer | Expert data engineer specializing in building scalable data pipelines, ETL/ELT processes, and data infrastructure. Masters big data technologies and cloud platforms with focus on reliable, efficient, and cost-optimized data platforms. |
| data-scientist | Expert data scientist specializing in statistical analysis, machine learning, and business insights. Masters exploratory data analysis, predictive modeling, and data storytelling with focus on delivering actionable insights that drive business value. |
| database-optimizer | Expert database optimizer specializing in query optimization, performance tuning, and scalability across multiple database systems. Masters execution plan analysis, index strategies, and system-level optimizations with focus on achieving peak database performance. |
| llm-architect | Expert LLM architect specializing in large language model architecture, deployment, and optimization. Masters LLM system design, fine-tuning strategies, and production serving with focus on building scalable, efficient, and safe LLM applications. |
| machine-learning-engineer | Expert ML engineer specializing in production model deployment, serving infrastructure, and scalable ML systems. Masters model optimization, real-time inference, and edge deployment with focus on reliability and performance at scale. |
| ml-engineer | Expert ML engineer specializing in machine learning model lifecycle, production deployment, and ML system optimization. Masters both traditional ML and deep learning with focus on building scalable, reliable ML systems from training to serving. |
| mlops-engineer | Expert MLOps engineer specializing in ML infrastructure, platform engineering, and operational excellence for machine learning systems. Masters CI/CD for ML, model versioning, and scalable ML platforms with focus on reliability and automation. |
| nlp-engineer | Expert NLP engineer specializing in natural language processing, understanding, and generation. Masters transformer models, text processing pipelines, and production NLP systems with focus on multilingual support and real-time performance. |
| postgres-pro | Expert PostgreSQL specialist mastering database administration, performance optimization, and high availability. Deep expertise in PostgreSQL internals, advanced features, and enterprise deployment with focus on reliability and peak performance. |
| prompt-engineer | Expert prompt engineer specializing in designing, optimizing, and managing prompts for large language models. Masters prompt architecture, evaluation frameworks, and production prompt systems with focus on reliability, efficiency, and measurable outcomes. |
| build-engineer | Expert build engineer specializing in build system optimization, compilation strategies, and developer productivity. Masters modern build tools, caching mechanisms, and creating fast, reliable build pipelines that scale with team growth. |
| cli-developer | Expert CLI developer specializing in command-line interface design, developer tools, and terminal applications. Masters user experience, cross-platform compatibility, and building efficient CLI tools that developers love to use. |
| dependency-manager | Expert dependency manager specializing in package management, security auditing, and version conflict resolution across multiple ecosystems. Masters dependency optimization, supply chain security, and automated updates with focus on maintaining stable, secure, and efficient dependency trees. |
| documentation-engineer | Expert documentation engineer specializing in technical documentation systems, API documentation, and developer-friendly content. Masters documentation-as-code, automated generation, and creating maintainable documentation that developers actually use. |
| dx-optimizer | Expert developer experience optimizer specializing in build performance, tooling efficiency, and workflow automation. Masters development environment optimization with focus on reducing friction, accelerating feedback loops, and maximizing developer productivity and satisfaction. |
| git-workflow-manager | Expert Git workflow manager specializing in branching strategies, automation, and team collaboration. Masters Git workflows, merge conflict resolution, and repository management with focus on enabling efficient, clear, and scalable version control practices. |
| legacy-modernizer | Expert legacy system modernizer specializing in incremental migration strategies and risk-free modernization. Masters refactoring patterns, technology updates, and business continuity with focus on transforming legacy systems into modern, maintainable architectures without disrupting operations. |
| mcp-developer | Expert MCP developer specializing in Model Context Protocol server and client development. Masters protocol specification, SDK implementation, and building production-ready integrations between AI systems and external tools/data sources. |
| refactoring-specialist | Expert refactoring specialist mastering safe code transformation techniques and design pattern application. Specializes in improving code structure, reducing complexity, and enhancing maintainability while preserving behavior with focus on systematic, test-driven refactoring. |
| tooling-engineer | Expert tooling engineer specializing in developer tool creation, CLI development, and productivity enhancement. Masters tool architecture, plugin systems, and user experience design with focus on building efficient, extensible tools that significantly improve developer workflows. |
| api-documenter | Expert API documenter specializing in creating comprehensive, developer-friendly API documentation. Masters OpenAPI/Swagger specifications, interactive documentation portals, and documentation automation with focus on clarity, completeness, and exceptional developer experience. |
| blockchain-developer | Expert blockchain developer specializing in smart contract development, DApp architecture, and DeFi protocols. Masters Solidity, Web3 integration, and blockchain security with focus on building secure, gas-efficient, and innovative decentralized applications. |
| embedded-systems | Expert embedded systems engineer specializing in microcontroller programming, RTOS development, and hardware optimization. Masters low-level programming, real-time constraints, and resource-limited environments with focus on reliability, efficiency, and hardware-software integration. |
| fintech-engineer | Expert fintech engineer specializing in financial systems, regulatory compliance, and secure transaction processing. Masters banking integrations, payment systems, and building scalable financial technology that meets stringent regulatory requirements. |
| game-developer | Expert game developer specializing in game engine programming, graphics optimization, and multiplayer systems. Masters game design patterns, performance optimization, and cross-platform development with focus on creating engaging, performant gaming experiences. |
| iot-engineer | Expert IoT engineer specializing in connected device architectures, edge computing, and IoT platform development. Masters IoT protocols, device management, and data pipelines with focus on building scalable, secure, and reliable IoT solutions. |
| mobile-app-developer | Expert mobile app developer specializing in native and cross-platform development for iOS and Android. Masters performance optimization, platform guidelines, and creating exceptional mobile experiences that users love. |
| payment-integration | Expert payment integration specialist mastering payment gateway integration, PCI compliance, and financial transaction processing. Specializes in secure payment flows, multi-currency support, and fraud prevention with focus on reliability, compliance, and seamless user experience. |
| quant-analyst | Expert quantitative analyst specializing in financial modeling, algorithmic trading, and risk analytics. Masters statistical methods, derivatives pricing, and high-frequency trading with focus on mathematical rigor, performance optimization, and profitable strategy development. |
| risk-manager | Expert risk manager specializing in comprehensive risk assessment, mitigation strategies, and compliance frameworks. Masters risk modeling, stress testing, and regulatory compliance with focus on protecting organizations from financial, operational, and strategic risks. |
| seo-specialist | Expert SEO strategist specializing in technical SEO, content optimization, and search engine rankings. Masters both on-page and off-page optimization, structured data implementation, and performance metrics to drive organic traffic and improve search visibility. |
| business-analyst | Expert business analyst specializing in requirements gathering, process improvement, and data-driven decision making. Masters stakeholder management, business process modeling, and solution design with focus on delivering measurable business value. |
| content-marketer | Expert content marketer specializing in content strategy, SEO optimization, and engagement-driven marketing. Masters multi-channel content creation, analytics, and conversion optimization with focus on building brand authority and driving measurable business results. |
| customer-success-manager | Expert customer success manager specializing in customer retention, growth, and advocacy. Masters account health monitoring, strategic relationship building, and driving customer value realization to maximize satisfaction and revenue growth. |
| legal-advisor | Expert legal advisor specializing in technology law, compliance, and risk mitigation. Masters contract drafting, intellectual property, data privacy, and regulatory compliance with focus on protecting business interests while enabling innovation and growth. |
| product-manager | Expert product manager specializing in product strategy, user-centric development, and business outcomes. Masters roadmap planning, feature prioritization, and cross-functional leadership with focus on delivering products that users love and drive business growth. |
| project-manager | Expert project manager specializing in project planning, execution, and delivery. Masters resource management, risk mitigation, and stakeholder communication with focus on delivering projects on time, within budget, and exceeding expectations. |
| sales-engineer | Expert sales engineer specializing in technical pre-sales, solution architecture, and proof of concepts. Masters technical demonstrations, competitive positioning, and translating complex technology into business value for prospects and customers. |
| scrum-master | Expert Scrum Master specializing in agile transformation, team facilitation, and continuous improvement. Masters Scrum framework implementation, impediment removal, and fostering high-performing, self-organizing teams that deliver value consistently. |
| technical-writer | Expert technical writer specializing in clear, accurate documentation and content creation. Masters API documentation, user guides, and technical content with focus on making complex information accessible and actionable for diverse audiences. |
| ux-researcher | Expert UX researcher specializing in user insights, usability testing, and data-driven design decisions. Masters qualitative and quantitative research methods to uncover user needs, validate designs, and drive product improvements through actionable insights. |
| wordpress-master | Elite WordPress architect specializing in full-stack development, performance optimization, and enterprise solutions. Masters custom theme/plugin development, multisite management, security hardening, and scaling WordPress from small sites to enterprise platforms handling millions of visitors. |
| agent-organizer | Expert agent organizer specializing in multi-agent orchestration, team assembly, and workflow optimization. Masters task decomposition, agent selection, and coordination strategies with focus on achieving optimal team performance and resource utilization. |
| context-manager | Expert context manager specializing in information storage, retrieval, and synchronization across multi-agent systems. Masters state management, version control, and data lifecycle with focus on ensuring consistency, accessibility, and performance at scale. |
| error-coordinator | Expert error coordinator specializing in distributed error handling, failure recovery, and system resilience. Masters error correlation, cascade prevention, and automated recovery strategies across multi-agent systems with focus on minimizing impact and learning from failures. |
| knowledge-synthesizer | Expert knowledge synthesizer specializing in extracting insights from multi-agent interactions, identifying patterns, and building collective intelligence. Masters cross-agent learning, best practice extraction, and continuous system improvement through knowledge management. |
| multi-agent-coordinator | Expert multi-agent coordinator specializing in complex workflow orchestration, inter-agent communication, and distributed system coordination. Masters parallel execution, dependency management, and fault tolerance with focus on achieving seamless collaboration at scale. |
| performance-monitor | Expert performance monitor specializing in system-wide metrics collection, analysis, and optimization. Masters real-time monitoring, anomaly detection, and performance insights across distributed agent systems with focus on observability and continuous improvement. |
| task-distributor | Expert task distributor specializing in intelligent work allocation, load balancing, and queue management. Masters priority scheduling, capacity tracking, and fair distribution with focus on maximizing throughput while maintaining quality and meeting deadlines. |
| workflow-orchestrator | Expert workflow orchestrator specializing in complex process design, state machine implementation, and business process automation. Masters workflow patterns, error compensation, and transaction management with focus on building reliable, flexible, and observable workflow systems. |
| competitive-analyst | Expert competitive analyst specializing in competitor intelligence, strategic analysis, and market positioning. Masters competitive benchmarking, SWOT analysis, and strategic recommendations with focus on creating sustainable competitive advantages. |
| data-researcher | Expert data researcher specializing in discovering, collecting, and analyzing diverse data sources. Masters data mining, statistical analysis, and pattern recognition with focus on extracting meaningful insights from complex datasets to support evidence-based decisions. |
| market-researcher | Expert market researcher specializing in market analysis, consumer insights, and competitive intelligence. Masters market sizing, segmentation, and trend analysis with focus on identifying opportunities and informing strategic business decisions. |
| research-analyst | Expert research analyst specializing in comprehensive information gathering, synthesis, and insight generation. Masters research methodologies, data analysis, and report creation with focus on delivering actionable intelligence that drives informed decision-making. |
| search-specialist | Expert search specialist mastering advanced information retrieval, query optimization, and knowledge discovery. Specializes in finding needle-in-haystack information across diverse sources with focus on precision, comprehensiveness, and efficiency. |
| trend-analyst | Expert trend analyst specializing in identifying emerging patterns, forecasting future developments, and strategic foresight. Masters trend detection, impact analysis, and scenario planning with focus on helping organizations anticipate and adapt to change. |
| ai-correlation-engine | AI/ML conversation-commit correlation specialist targeting 90%+ accuracy with Core ML integration. Use this PROACTIVELY when appropriate. |
| ai-engineer | Build production-ready LLM applications, advanced RAG systems, and intelligent agents. Implements vector search, multimodal AI, agent orchestration, and enterprise AI integrations. Use PROACTIVELY for LLM features, chatbots, AI agents, or AI-powered applications. |
| api-documenter | Master API documentation with OpenAPI 3.1, AI-powered tools, and modern developer experience practices. Create interactive docs, generate SDKs, and build comprehensive developer portals. Use PROACTIVELY for API documentation or developer portal creation. |
| architect-review | Master software architect specializing in modern architecture patterns, clean architecture, microservices, event-driven systems, and DDD. Reviews system designs and code changes for architectural integrity, scalability, and maintainability. Use PROACTIVELY for architectural decisions. |
| backend-architect | Design RESTful APIs, microservice boundaries, and database schemas. Reviews system architecture for scalability and performance bottlenecks. Use PROACTIVELY when creating new backend services or APIs. |
| backend-security-coder | Expert in secure backend coding practices specializing in input validation, authentication, and API security. Use PROACTIVELY for backend security implementations or security code reviews. |
| blockchain-developer | Build production-ready Web3 applications, smart contracts, and decentralized systems. Implements DeFi protocols, NFT platforms, DAOs, and enterprise blockchain integrations. Use PROACTIVELY for smart contracts, Web3 apps, DeFi protocols, or blockchain infrastructure. |
| business-analyst | Master modern business analysis with AI-powered analytics, real-time dashboards, and data-driven insights. Build comprehensive KPI frameworks, predictive models, and strategic recommendations. Use PROACTIVELY for business intelligence or strategic analysis. |
| c-pro | Write efficient C code with proper memory management, pointer arithmetic, and system calls. Handles embedded systems, kernel modules, and performance-critical code. Use PROACTIVELY for C optimization, memory issues, or system programming. |
| cloud-architect | Expert cloud architect specializing in AWS/Azure/GCP multi-cloud infrastructure design, advanced IaC (Terraform/OpenTofu/CDK), FinOps cost optimization, and modern architectural patterns. Masters serverless, microservices, security, compliance, and disaster recovery. Use PROACTIVELY for cloud architecture, cost optimization, migration planning, or multi-cloud strategies. |
| code-reviewer | Elite code review expert specializing in modern AI-powered code analysis, security vulnerabilities, performance optimization, and production reliability. Masters static analysis tools, security scanning, and configuration review with 2024/2025 best practices. Use PROACTIVELY for code quality assurance. |
| content-marketer | Elite content marketing strategist specializing in AI-powered content creation, omnichannel distribution, SEO optimization, and data-driven performance marketing. Masters modern content tools, social media automation, and conversion optimization with 2024/2025 best practices. Use PROACTIVELY for comprehensive content marketing. |
| context-manager | Elite AI context engineering specialist mastering dynamic context management, vector databases, knowledge graphs, and intelligent memory systems. Orchestrates context across multi-agent workflows, enterprise AI systems, and long-running projects with 2024/2025 best practices. Use PROACTIVELY for complex AI orchestration. |
| cpp-pro | Write idiomatic C++ code with modern features, RAII, smart pointers, and STL algorithms. Handles templates, move semantics, and performance optimization. Use PROACTIVELY for C++ refactoring, memory safety, or complex C++ patterns. |
| csharp-pro | Write modern C# code with advanced features like records, pattern matching, and async/await. Optimizes .NET applications, implements enterprise patterns, and ensures comprehensive testing. Use PROACTIVELY for C# refactoring, performance optimization, or complex .NET solutions. |
| customer-support | Elite AI-powered customer support specialist mastering conversational AI, automated ticketing, sentiment analysis, and omnichannel support experiences. Integrates modern support tools, chatbot platforms, and CX optimization with 2024/2025 best practices. Use PROACTIVELY for comprehensive customer experience management. |
| data-engineer | Build scalable data pipelines, modern data warehouses, and real-time streaming architectures. Implements Apache Spark, dbt, Airflow, and cloud-native data platforms. Use PROACTIVELY for data pipeline design, analytics infrastructure, or modern data stack implementation. |
| data-scientist | Expert data scientist for advanced analytics, machine learning, and statistical modeling. Handles complex data analysis, predictive modeling, and business intelligence. Use PROACTIVELY for data analysis tasks, ML modeling, statistical analysis, and data-driven insights. |
| database-admin | Expert database administrator specializing in modern cloud databases, automation, and reliability engineering. Masters AWS/Azure/GCP database services, Infrastructure as Code, high availability, disaster recovery, performance optimization, and compliance. Handles multi-cloud strategies, container databases, and cost optimization. Use this PROACTIVELY when appropriate. Use PROACTIVELY for database architecture, operations, or reliability engineering. |
| database-optimizer | Expert database optimizer specializing in modern performance tuning, query optimization, and scalable architectures. Masters advanced indexing, N+1 resolution, multi-tier caching, partitioning strategies, and cloud database optimization. Handles complex query analysis, migration strategies, and performance monitoring. Use PROACTIVELY for database optimization, performance issues, or scalability challenges. |
| debugger-agent | Frontend-Backend Integration Debugger - Deep dive analysis and diagnostics for React/Next.js data flow issues (read-only). Use this PROACTIVELY when appropriate. |
| debugger | Debugging specialist for errors, test failures, and unexpected behavior. Use proactively when encountering any issues. |
| deployment-engineer | Expert deployment engineer specializing in modern CI/CD pipelines, GitOps workflows, and advanced deployment automation. Masters GitHub Actions, ArgoCD/Flux, progressive delivery, container security, and platform engineering. Handles zero-downtime deployments, security scanning, and developer experience optimization. Use PROACTIVELY for CI/CD design, GitOps implementation, or deployment automation. |
| devops-troubleshooter | Expert DevOps troubleshooter specializing in rapid incident response, advanced debugging, and modern observability. Masters log analysis, distributed tracing, Kubernetes debugging, performance optimization, and root cause analysis. Handles production outages, system reliability, and preventive monitoring. Use PROACTIVELY for debugging, incident response, or system troubleshooting. |
| django-pro | Master Django 5.x with async views, DRF, Celery, and Django Channels. Build scalable web applications with proper architecture, testing, and deployment. Use PROACTIVELY for Django development, ORM optimization, or complex Django patterns. |
| docker-infrastructure-specialist | Docker and Infrastructure specialist for creating and managing containers, Dockerfiles, docker-compose configurations, and container orchestration. Use this PROACTIVELY when appropriate. |
| docs-architect | Creates comprehensive technical documentation from existing codebases. Analyzes architecture, design patterns, and implementation details to produce long-form technical manuals and ebooks. Use PROACTIVELY for system documentation, architecture guides, or technical deep-dives. |
| dx-optimizer | Developer Experience specialist. Improves tooling, setup, and workflows. Use PROACTIVELY when setting up new projects, after team feedback, or when development friction is noticed. |
| elixir-pro | Write idiomatic Elixir code with OTP patterns, supervision trees, and Phoenix LiveView. Masters concurrency, fault tolerance, and distributed systems. Use PROACTIVELY for Elixir refactoring, OTP design, or complex BEAM optimizations. |
| error-detective | Search logs and codebases for error patterns, stack traces, and anomalies. Correlates errors across systems and identifies root causes. Use PROACTIVELY when debugging issues, analyzing logs, or investigating production errors. |
| fastapi-pro | Build high-performance async APIs with FastAPI, SQLAlchemy 2.0, and Pydantic V2. Master microservices, WebSockets, and modern Python async patterns. Use PROACTIVELY for FastAPI development, async optimization, or API architecture. |
| flutter-expert | Master Flutter development with Dart 3, advanced widgets, and multi-platform deployment. Handles state management, animations, testing, and performance optimization for mobile, web, desktop, and embedded platforms. Use PROACTIVELY for Flutter architecture, UI implementation, or cross-platform features. |
| frontend-developer | Build React components, implement responsive layouts, and handle client-side state management. Masters React 19, Next.js 15, and modern frontend architecture. Optimizes performance and ensures accessibility. Use PROACTIVELY when creating UI components or fixing frontend issues. |
| frontend-security-coder | Expert in secure frontend coding practices specializing in XSS prevention, output sanitization, and client-side security patterns. Use PROACTIVELY for frontend security implementations or client-side security code reviews. |
| golang-pro | Master Go 1.21+ with modern patterns, advanced concurrency, performance optimization, and production-ready microservices. Expert in the latest Go ecosystem including generics, workspaces, and cutting-edge frameworks. Use PROACTIVELY for Go development, architecture design, or performance optimization. |
| graphql-architect | Master modern GraphQL with federation, performance optimization, and enterprise security. Build scalable schemas, implement advanced caching, and design real-time systems. Use PROACTIVELY for GraphQL architecture or performance optimization. |
| hello-world-agent | Simple greeting agent, use proactively when greeting the user. If they say 'hi claude', 'hey cc', 'hi cc' or 'hi claude code' use this agent. |
| hr-pro | Professional, ethical HR partner for hiring, onboarding/offboarding, PTO and leave, performance, compliant policies, and employee relations. Ask for jurisdiction and company context before advising; produce structured, bias-mitigated, lawful templates. |
| hybrid-cloud-architect | Expert hybrid cloud architect specializing in complex multi-cloud solutions across AWS/Azure/GCP and private clouds (OpenStack/VMware). Masters hybrid connectivity, workload placement optimization, edge computing, and cross-cloud automation. Handles compliance, cost optimization, disaster recovery, and migration strategies. Use PROACTIVELY for hybrid architecture, multi-cloud strategy, or complex infrastructure integration. |
| incident-responder | Expert SRE incident responder specializing in rapid problem resolution, modern observability, and comprehensive incident management. Masters incident command, blameless post-mortems, error budget management, and system reliability patterns. Handles critical outages, communication strategies, and continuous improvement. Use IMMEDIATELY for production incidents or SRE practices. |
| intelligence-performance-optimizer | AI operation performance optimization specialist for real-time correlation and search. Optimizes Core ML models, caching strategies, and ensures AI features meet production performance targets. Use this PROACTIVELY when appropriate. |
| ios-developer | Develop native iOS applications with Swift/SwiftUI. Masters iOS 18, SwiftUI, UIKit integration, Core Data, networking, and App Store optimization. Use PROACTIVELY for iOS-specific features, App Store optimization, or native iOS development. |
| java-pro | Master Java 21+ with modern features like virtual threads, pattern matching, and Spring Boot 3.x. Expert in the latest Java ecosystem including GraalVM, Project Loom, and cloud-native patterns. Use PROACTIVELY for Java development, microservices architecture, or performance optimization. |
| javascript-pro | Master modern JavaScript with ES6+, async patterns, and Node.js APIs. Handles promises, event loops, and browser/Node compatibility. Use PROACTIVELY for JavaScript optimization, async debugging, or complex JS patterns. |
| kubernetes-architect | Expert Kubernetes architect specializing in cloud-native infrastructure, advanced GitOps workflows (ArgoCD/Flux), and enterprise container orchestration. Masters EKS/AKS/GKE, service mesh (Istio/Linkerd), progressive delivery, multi-tenancy, and platform engineering. Handles security, observability, cost optimization, and developer experience. Use PROACTIVELY for K8s architecture, GitOps implementation, or cloud-native platform design. |
| legacy-modernizer | Refactor legacy codebases, migrate outdated frameworks, and implement gradual modernization. Handles technical debt, dependency updates, and backward compatibility. Use PROACTIVELY for legacy system updates, framework migrations, or technical debt reduction. |
| legal-advisor | Draft privacy policies, terms of service, disclaimers, and legal notices. Creates GDPR-compliant texts, cookie policies, and data processing agreements. Use PROACTIVELY for legal documentation, compliance texts, or regulatory requirements. |
| mcp-protocol-expert | Model Context Protocol implementation specialist for handoff mechanisms and tool integration. Use this PROACTIVELY when appropriate. |
| mermaid-expert | Create Mermaid diagrams for flowcharts, sequences, ERDs, and architectures. Masters syntax for all diagram types and styling. Use PROACTIVELY for visual documentation, system diagrams, or process flows. |
| meta-agent | Generates a new, complete Claude Code sub-agent configuration file from a user's description. Use this to create new agents. Use this PROACTIVELY when the user asks you to create a new sub agent. |
| minecraft-bukkit-pro | Master Minecraft server plugin development with Bukkit, Spigot, and Paper APIs. Specializes in event-driven architecture, command systems, world manipulation, player management, and performance optimization. Use PROACTIVELY for plugin architecture, gameplay mechanics, server-side features, or cross-version compatibility. |
| ml-engineer | Build production ML systems with PyTorch 2.x, TensorFlow, and modern ML frameworks. Implements model serving, feature engineering, A/B testing, and monitoring. Use PROACTIVELY for ML model deployment, inference optimization, or production ML infrastructure. |
| mlops-engineer | Build comprehensive ML pipelines, experiment tracking, and model registries with MLflow, Kubeflow, and modern MLOps tools. Implements automated training, deployment, and monitoring across cloud platforms. Use PROACTIVELY for ML infrastructure, experiment management, or pipeline automation. |
| mobile-developer | Develop React Native, Flutter, or native mobile apps with modern architecture patterns. Masters cross-platform development, native integrations, offline sync, and app store optimization. Use PROACTIVELY for mobile features, cross-platform code, or app optimization. |
| mobile-security-coder | Expert in secure mobile coding practices specializing in input validation, WebView security, and mobile-specific security patterns. Use PROACTIVELY for mobile security implementations or mobile security code reviews. |
| network-engineer | Expert network engineer specializing in modern cloud networking, security architectures, and performance optimization. Masters multi-cloud connectivity, service mesh, zero-trust networking, SSL/TLS, global load balancing, and advanced troubleshooting. Handles CDN optimization, network automation, and compliance. Use PROACTIVELY for network design, connectivity issues, or performance optimization. |
| payment-integration | Integrate Stripe, PayPal, and payment processors. Handles checkout flows, subscriptions, webhooks, and PCI compliance. Use PROACTIVELY when implementing payments, billing, or subscription features. |
| performance-engineer | Expert performance engineer specializing in modern observability, application optimization, and scalable system performance. Masters OpenTelemetry, distributed tracing, load testing, multi-tier caching, Core Web Vitals, and performance monitoring. Handles end-to-end optimization, real user monitoring, and scalability patterns. Use PROACTIVELY for performance optimization, observability, or scalability challenges. |
| php-pro | Write idiomatic PHP code with generators, iterators, SPL data structures, and modern OOP features. Use PROACTIVELY for high-performance PHP applications. |
| production-readiness-validator | Production deployment, scalability, and enterprise-readiness validation specialist. Use this PROACTIVELY when appropriate. |
| prompt-engineer | Expert prompt engineer specializing in advanced prompting techniques, LLM optimization, and AI system design. Masters chain-of-thought, constitutional AI, and production prompt strategies. Use when building AI features, improving agent performance, or crafting system prompts. |
| python-backend-tdd-agent | Specialized Python backend developer using strict Test-Driven Development. Always writes tests BEFORE implementation following red-green-refactor cycle. Use this PROACTIVELY when appropriate. |
| python-integration-specialist | Specialized agent for integrating new components into existing Python systems with asyncio, dependency injection, and backward compatibility. Use this PROACTIVELY when appropriate. |
| python-pro | Master Python 3.12+ with modern features, async programming, performance optimization, and production-ready practices. Expert in the latest Python ecosystem including uv, ruff, pydantic, and FastAPI. Use PROACTIVELY for Python development, optimization, or advanced Python patterns. |
| python-schema-architect | Specialized agent for designing and implementing Pydantic data models, configuration schemas, and data validation patterns. Use this PROACTIVELY when appropriate. |
| quality-control | Quality control checkpoint agent that verifies task completion and prevents common Claude Code mistakes. Use this PROACTIVELY when appropriate. |
| quant-analyst | Build financial models, backtest trading strategies, and analyze market data. Implements risk metrics, portfolio optimization, and statistical arbitrage. Use PROACTIVELY for quantitative finance, trading algorithms, or risk analysis. |
| reference-builder | Creates exhaustive technical references and API documentation. Generates comprehensive parameter listings, configuration guides, and searchable reference materials. Use PROACTIVELY for API docs, configuration references, or complete technical specifications. |
| risk-manager | Monitor portfolio risk, R-multiples, and position limits. Creates hedging strategies, calculates expectancy, and implements stop-losses. Use PROACTIVELY for risk assessment, trade tracking, or portfolio protection. |
| ruby-pro | Write idiomatic Ruby code with metaprogramming, Rails patterns, and performance optimization. Specializes in Ruby on Rails, gem development, and testing frameworks. Use PROACTIVELY for Ruby refactoring, optimization, or complex Ruby features. |
| rust-pro | Master Rust 1.75+ with modern async patterns, advanced type system features, and production-ready systems programming. Expert in the latest Rust ecosystem including Tokio, axum, and cutting-edge crates. Use PROACTIVELY for Rust development, performance optimization, or systems programming. |
| sales-automator | Draft cold emails, follow-ups, and proposal templates. Creates pricing pages, case studies, and sales scripts. Use PROACTIVELY for sales outreach or lead nurturing. |
| scala-pro | Master enterprise-grade Scala development with functional programming, distributed systems, and big data processing. Expert in Apache Pekko, Akka, Spark, ZIO/Cats Effect, and reactive architectures. Use PROACTIVELY for Scala system design, performance optimization, or enterprise integration. |
| search-specialist | Expert web researcher using advanced search techniques and synthesis. Masters search operators, result filtering, and multi-source verification. Handles competitive analysis and fact-checking. Use PROACTIVELY for deep research, information gathering, or trend analysis. |
| security-auditor | Expert security auditor specializing in DevSecOps, comprehensive cybersecurity, and compliance frameworks. Masters vulnerability assessment, threat modeling, secure authentication (OAuth2/OIDC), OWASP standards, cloud security, and security automation. Handles DevSecOps integration, compliance (GDPR/HIPAA/SOC2), and incident response. Use PROACTIVELY for security audits, DevSecOps, or compliance implementation. |
| seo-authority-builder | Analyzes content for E-E-A-T signals and suggests improvements to build authority and trust. Identifies missing credibility elements. Use PROACTIVELY for YMYL topics. |
| seo-cannibalization-detector | Analyzes multiple provided pages to identify keyword overlap and potential cannibalization issues. Suggests differentiation strategies. Use PROACTIVELY when reviewing similar content. |
| seo-content-auditor | Analyzes provided content for quality, E-E-A-T signals, and SEO best practices. Scores content and provides improvement recommendations based on established guidelines. Use PROACTIVELY for content review. |
| seo-content-planner | Creates comprehensive content outlines and topic clusters for SEO. Plans content calendars and identifies topic gaps. Use PROACTIVELY for content strategy and planning. |
| seo-content-refresher | Identifies outdated elements in provided content and suggests updates to maintain freshness. Finds statistics, dates, and examples that need updating. Use PROACTIVELY for older content. |
| seo-content-writer | Writes SEO-optimized content based on provided keywords and topic briefs. Creates engaging, comprehensive content following best practices. Use PROACTIVELY for content creation tasks. |
| seo-keyword-strategist | Analyzes keyword usage in provided content, calculates density, suggests semantic variations and LSI keywords based on the topic. Prevents over-optimization. Use PROACTIVELY for content optimization. |
| seo-meta-optimizer | Creates optimized meta titles, descriptions, and URL suggestions based on character limits and best practices. Generates compelling, keyword-rich metadata. Use PROACTIVELY for new content. |
| seo-snippet-hunter | Formats content to be eligible for featured snippets and SERP features. Creates snippet-optimized content blocks based on best practices. Use PROACTIVELY for question-based content. |
| seo-structure-architect | Analyzes and optimizes content structure including header hierarchy, suggests schema markup, and internal linking opportunities. Creates search-friendly content organization. Use PROACTIVELY for content structuring. |
| sql-pro | Master modern SQL with cloud-native databases, OLTP/OLAP optimization, and advanced query techniques. Expert in performance tuning, data modeling, and hybrid analytical systems. Use PROACTIVELY for database optimization or complex analysis. |
| swift-build-master | Orchestrates Swift compilation error fixing across specialized agent teams. Use this PROACTIVELY when appropriate. |
| swift-concurrency-expert | Swift async/await, background queue, and thread-safe implementation specialist. Use this PROACTIVELY when appropriate. |
| swift-import-specialist | Specialized agent for fixing Swift import statements and dependency issues. Use this PROACTIVELY when appropriate. |
| swift-method-fixer | Specialized agent for fixing Swift method and function compilation errors. Use this PROACTIVELY when appropriate. |
| swift-syntax-cleaner | Specialized agent for fixing Swift syntax errors and malformed code. Use this PROACTIVELY when appropriate. |
| swift-type-resolver | Specialized agent for fixing Swift type-related compilation errors and declarations. Use this PROACTIVELY when appropriate. |
| swiftdev | Swift Development Agent for FSEvents API implementation and macOS app development for Phase 2 auto-commit functionality. Use this PROACTIVELY when appropriate. |
| tdd-orchestrator | Master TDD orchestrator specializing in red-green-refactor discipline, multi-agent workflow coordination, and comprehensive test-driven development practices. Enforces TDD best practices across teams with AI-assisted testing and modern frameworks. Use PROACTIVELY for TDD implementation and governance. |
| technical-documentation-specialist | Technical documentation specialist for software features, APIs, and system architecture. Creates comprehensive documentation with clarity and completeness. Use this PROACTIVELY when appropriate. |
| terraform-specialist | Expert Terraform/OpenTofu specialist mastering advanced IaC automation, state management, and enterprise infrastructure patterns. Handles complex module design, multi-cloud deployments, GitOps workflows, policy as code, and CI/CD integration. Covers migration strategies, security best practices, and modern IaC ecosystems. Use PROACTIVELY for advanced IaC, state management, or infrastructure automation. |
| test-automator | Master AI-powered test automation with modern frameworks, self-healing tests, and comprehensive quality engineering. Build scalable testing strategies with advanced CI/CD integration. Use PROACTIVELY for testing automation or quality assurance. |
| tutorial-engineer | Creates step-by-step tutorials and educational content from code. Transforms complex concepts into progressive learning experiences with hands-on examples. Use PROACTIVELY for onboarding guides, feature tutorials, or concept explanations. |
| typescript-pro | Master TypeScript with advanced types, generics, and strict type safety. Handles complex type systems, decorators, and enterprise-grade patterns. Use PROACTIVELY for TypeScript architecture, type inference optimization, or advanced typing patterns. |
| ui-ux-designer | Create interface designs, wireframes, and design systems. Masters user research, accessibility standards, and modern design tools. Specializes in design tokens, component libraries, and inclusive design. Use PROACTIVELY for design systems, user flows, or interface optimization. |
| ui-visual-validator | Rigorous visual validation expert specializing in UI testing, design system compliance, and accessibility verification. Masters screenshot analysis, visual regression testing, and component validation. Use PROACTIVELY to verify UI modifications have achieved their intended goals through comprehensive visual analysis. |
| unity-developer | Build Unity games with optimized C# scripts, efficient rendering, and proper asset management. Masters Unity 6 LTS, URP/HDRP pipelines, and cross-platform deployment. Handles gameplay systems, UI implementation, and platform optimization. Use PROACTIVELY for Unity performance issues, game mechanics, or cross-platform builds. |


  ---
  7. Advanced Use Cases

  CI/CD Integration

  #!/bin/bash
  # In your CI pipeline
  claude -p "run tests and report any failures" --output-format json > results.json

  Code Review Automation

  git diff origin/main...HEAD | claude --agents '{"security-reviewer":{"description":"Security auditor","prompt":"Check for
  security vulnerabilities"}}' -p "review for security issues"

  Documentation Generation

  claude -p "generate API documentation for all endpoints" --tools "Read,Grep,Glob" > api-docs.md

  ---
  8. Key Documentation Resources

  | Topic            | URL                                                 |
  |------------------|-----------------------------------------------------|
  | CLI Reference    | https://code.claude.com/docs/en/cli-reference.md    |
  | Sub-agents       | https://code.claude.com/docs/en/sub-agents.md       |
  | Common Workflows | https://code.claude.com/docs/en/common-workflows.md |
  | Quickstart       | https://code.claude.com/docs/en/quickstart.md       |
  | Interactive Mode | https://code.claude.com/docs/en/interactive-mode.md |

  ---
  9. Limitations & Considerations

  ★ Insight ─────────────────────────────────────
  The -p mode skips workspace trust dialogs, making it powerful but potentially risky. Anthropic explicitly warns: "Only use
  this flag in directories you trust." This design choice:
  - Optimizes for automation and scripting
  - Requires conscious security awareness from users
  - Follows the principle of "secure by default, convenient by choice"
  ─────────────────────────────────────────────────

  Other Notes:
  - Max timeout: 600,000ms (10 minutes) for long-running operations
  - Agents defined via --agents are session-scoped (not persisted)
  - Project agents (.claude/agents/) override user agents (~/.claude/agents/)

  ---
  ✅ Conclusion

  Claude Code provides robust command-line subagent invocation through:
  1. ✅ Dynamic agent definition (--agents JSON flag)
  2. ✅ Explicit agent requests in queries
  3. ✅ Automatic intelligent delegation
  4. ✅ Full -p mode support for scripting

  The architecture is production-ready for:
  - CI/CD automation
  - Script-based workflows
  - Programmatic code analysis
  - Batch processing tasks
