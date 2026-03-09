# Analysis: Survey of LLM Agent Communication with MCP

**Paper:** Survey of LLM Agent Communication with MCP: A Software Design Pattern Centric Review  
**Authors:** Anjana Sarkar, Soumyendu Sarkar  
**Rating:** 7/10

## What It Does

This paper maps classical software design patterns (Mediator, Observer, Pub/Sub) to LLM multi-agent systems, with MCP as the central protocol. It covers architectures (centralized, decentralized, hierarchical), provides financial services case studies, and positions MCP as the foundation for agent communication.

## Strengths

**Solid theoretical foundation** - The paper successfully connects traditional software engineering to modern AI agents. The mathematical models for communication overhead (O(N²) vs O(N)) are clear, and the design pattern mappings make sense.

**MCP positioning** - Frames MCP as "USB-C for AI" solving the N×M integration problem. Shows how it embodies multiple patterns simultaneously and fits into a broader protocol stack (MCP → ACP → A2A → ANP).

**Practical case studies** - Financial services examples (fraud detection, portfolio management, M&A due diligence) demonstrate real applications with specific pattern recommendations.

**Comprehensive scope** - Covers major frameworks (AutoGen, LangChain, CrewAI, MetaGPT), architectural patterns, security considerations, and future directions.

## Weaknesses

**No empirical validation** - This is the biggest issue. All claims are theoretical. No benchmarks, no performance data, no real measurements. The mathematical models aren't validated with actual systems.

**Missing implementation details** - Lots of concepts but no code. Practitioners can't easily translate these ideas into working systems. Need examples showing how to actually implement these patterns with MCP.

**MCP-centric without critical analysis** - Heavy focus on MCP's benefits but doesn't discuss limitations, failure modes, or scenarios where simpler approaches work better. Feels promotional.

**Shallow security coverage** - Mentions Agent-in-the-Middle attacks and OAuth but doesn't go deep. No threat modeling, attack surface analysis, or detailed security architecture.

**No evaluation framework** - Identifies the need for benchmarking but doesn't propose specific metrics. How do you measure if your multi-agent system is working well?

**Breadth over depth** - Covers many topics but some feel superficial. Financial services is detailed, but other domains (healthcare, legal) are barely touched.

## What's Missing

1. **Experiments** - Implement 2-3 patterns, measure performance, compare approaches
2. **Code examples** - Python snippets using MCP SDK, configuration templates, step-by-step guides
3. **Decision framework** - How do you choose which pattern to use? Need a flowchart or decision tree
4. **Security deep-dive** - Threat models, attack scenarios, mitigation strategies
5. **Metrics** - Define KPIs for multi-agent systems, propose benchmarking methodology

## Technical Notes

**Architecture taxonomy is useful:**
- Centralized: Good for <10 agents, tight control needed
- Decentralized: Scales better, more resilient, harder to debug
- Hierarchical: Complex task decomposition, clear specialization
- Hybrid: Adapts based on workload

**MCP coverage is decent but incomplete:**
- Explains client-host-server model well
- Doesn't discuss transport options (stdio vs HTTP/SSE)
- Missing scalability limits, server discovery, versioning

**Mathematical models need work:**
- Communication complexity formulas are clear
- Information entropy model is interesting
- But none are validated with real data

## Who Should Read This

**Good for:**
- Researchers wanting a literature review and theoretical foundation
- Enterprise architects needing high-level guidance
- Anyone trying to understand MCP's role in agent ecosystems

**Not ideal for:**
- Developers needing implementation guidance (too conceptual)
- Security engineers (not enough depth)
- Beginners (assumes background knowledge)

## Bottom Line

Decent survey that connects software engineering principles to LLM agents. The MCP focus is timely and the design pattern framework is useful. But it's mostly theoretical - needs empirical validation, code examples, and deeper security analysis to be truly valuable for practitioners.

The paper establishes a good conceptual foundation but stops short of proving these ideas work in practice. Add some experiments and implementation details, and this becomes much stronger.

**Recommendation:** Useful as a starting point for understanding agent communication architectures, but you'll need other resources to actually build something.
