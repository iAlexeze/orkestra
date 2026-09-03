# Why I Built This

A few years ago I went for an interview and didn't get the job.

The requirement was that the person knew how to write Kubernetes controllers and operators. Several of them. I knew Kubernetes. I knew how to deploy to it, how to operate it, how to think about it. But I had never written an operator from scratch, and that gap showed.

I decided I wasn't going to miss another opportunity because of that.

---

I started learning. I hit a wall — not the architecture, not the concept. The scaffolding. The boilerplate. The sheer amount of code that stood between me and a working operator. What I had assumed was an advanced skill turned out to require expert-level knowledge just to get started. Every tutorial I followed assumed I already knew client-go, informers, schemes, workqueues. Every framework I tried gave me a project structure I didn't fully understand and hundreds of lines of generated code I couldn't easily modify.

The idea was simple. The path to implementing the idea was not.

---

The further I got, the more I started noticing something. The reconcile loop was always the same shape. The informer setup was always the same. The queue depth, the retry logic, the event handling — identical across every operator I read. The business logic — the actual interesting part, the thing that made each operator useful — was a small fraction of the total code. Everything else was scaffolding that existed for its own sake.

I started thinking about that pattern. If the structure is always the same, why is everyone writing it again?

---

I started talking to engineers. Not to validate a product idea. Just because I had questions. What I heard was the same experience repeated in different words.

*"We needed six operators. We built one and ran out of time."*

*"I got the scaffolding running and then saw how much I still had to write."*

*"We just do it manually now."*

And from developers who just wanted to deploy their applications:

*"I just want my app running. I don't want to learn Kubernetes to do it."*

Nobody was describing an edge case. They were describing the default experience.

---

I kept building. I built operators differently each time — sometimes borrowing patterns from the last one, sometimes starting fresh. At some point I had enough examples in front of me that I stopped thinking about them as programs and started thinking about them as **behaviours**. Patterns. A deployment is a pattern. A service is a pattern. A secret generated once is a pattern. The operator is just the thing that applies those patterns when a CR appears and keeps applying them until the CR is gone.

If the behaviours are patterns, you only have to describe them once.

That insight was the whole thing. Once I saw it that way, everything else followed. You don't write a new reconciler for each CRD. You write a description of what a reconciler for that CRD should do — what resources it creates, when it creates them, what it watches, how it reports health. A runtime reads that description and does the work. Your job is the description, not the implementation.

That is what the Katalog is. A description of operator behaviour. And that is what the runtime is — the single thing that reads Katalogs and runs them.

---

I struggled with the name. It had to be something that communicated the idea without requiring explanation. Something that pointed at composition — at multiple things working together — without being generic or cold.

I kept coming back to music. An orchestra takes distinct instruments, each with its own voice, and composes them into something that no single instrument could produce. That is exactly what this does. You write Katalogs — individual operator descriptions, each focused on one concern. The runtime plays them together. The result is a platform.

**Orkestra.**

And once I had that frame, the rest of the naming came naturally. A **Motif** is the smallest reusable unit — a named set of resource patterns you can import into any operator, like a musical phrase you can carry across movements. A **Komposer** composes multiple Katalogs into a platform, the way a composer arranges instruments into a symphony. **Notes** are the template functions available inside every Katalog expression — the atomic building blocks, pure and combinable. The **Konductor** is the elected leader in a multi-replica deployment, the instance currently conducting the reconcile. The **Control Center** is the dashboard — the stage where you watch the performance, see which operators are healthy, which are reconciling, which need attention.

Every name is the same concept applied at a different scale. You write motifs. You import them into a Katalog. You compose Katalogs into a Komposer. The runtime konducts them. The control center shows you the symphony.

---

I built this because I missed an interview. I kept building it because every engineer I spoke to described the same wall. The goal was never to replace operators — operators are the right pattern. The goal was to make them something platform engineers and developers could reach without fear, without a week of scaffolding, without becoming an expert in client-go just to get started.

You should be able to describe what an operator does and have it run. That is what a declarative platform is supposed to feel like.

That is Orkestra. Declare. Run.
