# Unity Patterns Reference

## Contents

- [State Machine](#state-machine)
- [Animation Patterns](#animation-patterns)
- [Save System](#save-system)
- [Audio Manager](#audio-manager)
- [Networking (Netcode)](#networking-netcode)
- [Performance Optimization](#performance-optimization)
- [Editor Scripting](#editor-scripting)
- [Testing Patterns](#testing-patterns)
- [Advanced Object Pooling](#advanced-object-pooling)
- [Coroutine Patterns](#coroutine-patterns)

## State Machine

### Generic Finite State Machine

```csharp
public interface IState
{
    void Enter();
    void Execute();
    void Exit();
}

public class StateMachine
{
    private IState currentState;
    private readonly Dictionary<Type, IState> states = new();

    public void AddState(IState state) => states[state.GetType()] = state;

    public void SetState<T>() where T : IState
    {
        if (!states.TryGetValue(typeof(T), out var newState)) return;
        currentState?.Exit();
        currentState = newState;
        currentState.Enter();
    }

    public void Update() => currentState?.Execute();

    public bool IsInState<T>() where T : IState => currentState is T;
}
```

### Enemy AI States

```csharp
public class IdleState : IState
{
    private readonly EnemyAI enemy;
    private float waitTimer;

    public IdleState(EnemyAI enemy) => this.enemy = enemy;

    public void Enter() => waitTimer = enemy.Config.patrolWaitTime;

    public void Execute()
    {
        waitTimer -= Time.deltaTime;
        if (waitTimer <= 0f)
            enemy.StateMachine.SetState<PatrolState>();

        if (enemy.CanSeePlayer())
            enemy.StateMachine.SetState<ChaseState>();
    }

    public void Exit() { }
}

public class ChaseState : IState
{
    private readonly EnemyAI enemy;

    public ChaseState(EnemyAI enemy) => this.enemy = enemy;

    public void Enter() => enemy.Animator.SetBool("IsRunning", true);

    public void Execute()
    {
        var distance = Vector3.Distance(enemy.transform.position, enemy.Target.position);

        if (distance <= enemy.Config.attackRange)
            enemy.StateMachine.SetState<AttackState>();
        else if (distance > enemy.Config.chaseRange)
            enemy.StateMachine.SetState<IdleState>();
        else
            enemy.NavAgent.SetDestination(enemy.Target.position);
    }

    public void Exit() => enemy.Animator.SetBool("IsRunning", false);
}
```

### ScriptableObject-Based State Machine

```csharp
public abstract class StateData : ScriptableObject
{
    public abstract void OnEnter(EnemyAI enemy);
    public abstract void OnExecute(EnemyAI enemy);
    public abstract void OnExit(EnemyAI enemy);
}

[CreateAssetMenu(menuName = "Game/AI/Patrol State")]
public class PatrolStateData : StateData
{
    public float patrolRadius = 10f;
    public float patrolSpeed = 2f;

    public override void OnEnter(EnemyAI enemy)
    {
        enemy.NavAgent.speed = patrolSpeed;
        enemy.SetRandomPatrolPoint(patrolRadius);
    }

    public override void OnExecute(EnemyAI enemy)
    {
        if (enemy.NavAgent.remainingDistance < 0.5f)
            enemy.SetRandomPatrolPoint(patrolRadius);
    }

    public override void OnExit(EnemyAI enemy) { }
}
```

## Animation Patterns

### Animator Controller Wrapper

```csharp
public class CharacterAnimator : MonoBehaviour
{
    private Animator animator;

    // Cache parameter hashes (avoid string lookups every frame)
    private static readonly int SpeedHash = Animator.StringToHash("Speed");
    private static readonly int IsGroundedHash = Animator.StringToHash("IsGrounded");
    private static readonly int JumpHash = Animator.StringToHash("Jump");
    private static readonly int AttackHash = Animator.StringToHash("Attack");
    private static readonly int DieHash = Animator.StringToHash("Die");

    private void Awake() => animator = GetComponent<Animator>();

    public void SetSpeed(float speed) => animator.SetFloat(SpeedHash, speed);
    public void SetGrounded(bool grounded) => animator.SetBool(IsGroundedHash, grounded);
    public void TriggerJump() => animator.SetTrigger(JumpHash);
    public void TriggerAttack() => animator.SetTrigger(AttackHash);
    public void TriggerDie() => animator.SetTrigger(DieHash);
}
```

**Rules**: Always use `Animator.StringToHash` for parameter names (cached as static readonly). Never use string parameter names in Update loops. Create a wrapper class per character type to centralize animation control.

### Animation Events

```csharp
// Called from Animation Event keyframes in the Animation window
public class AttackAnimationEvents : MonoBehaviour
{
    [SerializeField] private Transform attackPoint;
    [SerializeField] private float attackRadius = 1f;
    [SerializeField] private int damage = 10;
    [SerializeField] private LayerMask targetLayers;

    // Referenced by name in Animation Event
    public void OnAttackHit()
    {
        var hits = Physics.OverlapSphere(attackPoint.position, attackRadius, targetLayers);
        foreach (var hit in hits)
        {
            if (hit.TryGetComponent<IDamageable>(out var damageable))
                damageable.TakeDamage(damage);
        }
    }

    public void OnAttackEnd()
    {
        // Reset attack state, allow next action
    }
}
```

### DOTween-Style Tweening (without dependency)

```csharp
public static class TweenHelper
{
    public static IEnumerator LerpPosition(
        Transform target, Vector3 end, float duration, System.Action onComplete = null)
    {
        var start = target.position;
        float elapsed = 0f;

        while (elapsed < duration)
        {
            elapsed += Time.deltaTime;
            float t = Mathf.SmoothStep(0f, 1f, elapsed / duration);
            target.position = Vector3.Lerp(start, end, t);
            yield return null;
        }

        target.position = end;
        onComplete?.Invoke();
    }

    public static IEnumerator FadeCanvasGroup(
        CanvasGroup group, float targetAlpha, float duration)
    {
        float startAlpha = group.alpha;
        float elapsed = 0f;

        while (elapsed < duration)
        {
            elapsed += Time.deltaTime;
            group.alpha = Mathf.Lerp(startAlpha, targetAlpha, elapsed / duration);
            yield return null;
        }

        group.alpha = targetAlpha;
        group.interactable = targetAlpha > 0f;
        group.blocksRaycasts = targetAlpha > 0f;
    }
}
```

## Save System

### JSON Save with Encryption Option

```csharp
public interface ISaveSystem
{
    void Save<T>(string key, T data);
    T Load<T>(string key, T defaultValue = default);
    bool HasSave(string key);
    void DeleteSave(string key);
}

[Serializable]
public struct Vector3Serializable
{
    public float x, y, z;
    public Vector3Serializable(Vector3 v) { x = v.x; y = v.y; z = v.z; }
    public Vector3 ToVector3() => new(x, y, z);
}

[Serializable]
public struct QuaternionSerializable
{
    public float x, y, z, w;
    public QuaternionSerializable(Quaternion q) { x = q.x; y = q.y; z = q.z; w = q.w; }
    public Quaternion ToQuaternion() => new(x, y, z, w);
}

public class SaveSystem : MonoBehaviour, ISaveSystem
{
    private string SavePath => Application.persistentDataPath;

    public void Save<T>(string key, T data)
    {
        try
        {
            var json = JsonUtility.ToJson(data, prettyPrint: true);
            var path = Path.Combine(SavePath, $"{key}.json");
            File.WriteAllText(path, json);
        }
        catch (Exception e)
        {
            Debug.LogError($"Save failed for {key}: {e.Message}");
        }
    }

    public T Load<T>(string key, T defaultValue = default)
    {
        var path = Path.Combine(SavePath, $"{key}.json");
        if (!File.Exists(path)) return defaultValue;

        try
        {
            var json = File.ReadAllText(path);
            return JsonUtility.FromJson<T>(json);
        }
        catch (Exception e)
        {
            Debug.LogError($"Load failed for {key}: {e.Message}");
            return defaultValue;
        }
    }

    public bool HasSave(string key) => File.Exists(Path.Combine(SavePath, $"{key}.json"));
    public void DeleteSave(string key)
    {
        var path = Path.Combine(SavePath, $"{key}.json");
        if (File.Exists(path)) File.Delete(path);
    }
}
```

**Important**: `JsonUtility` does not serialize dictionaries, polymorphic types, or properties. For complex data, use Newtonsoft.Json (via `com.unity.nuget.newtonsoft-json` package). Always validate paths to prevent directory traversal.

## Audio Manager

```csharp
public interface IAudioManager
{
    void PlaySFX(AudioClip clip, float volume = 1f);
    void PlaySFXAtPosition(AudioClip clip, Vector3 position, float volume = 1f);
    void PlayMusic(AudioClip clip, bool loop = true);
    void StopMusic();
    void SetVolume(string parameter, float volume);
}

public class AudioManager : MonoBehaviour, IAudioManager
{
    [SerializeField] private AudioMixer audioMixer;
    [SerializeField] private AudioSource musicSource;
    [SerializeField] private AudioSource sfxSource;
    [SerializeField] private int sfxPoolSize = 10;

    private readonly Queue<AudioSource> sfxPool = new();

    private void Awake()
    {
        for (int i = 0; i < sfxPoolSize; i++)
        {
            var source = gameObject.AddComponent<AudioSource>();
            source.playOnAwake = false;
            source.outputAudioMixerGroup = sfxSource.outputAudioMixerGroup;
            sfxPool.Enqueue(source);
        }
        LoadVolumeSettings();
    }

    public void PlaySFX(AudioClip clip, float volume = 1f)
    {
        if (clip != null) sfxSource.PlayOneShot(clip, volume);
    }

    public void PlaySFXAtPosition(AudioClip clip, Vector3 position, float volume = 1f)
    {
        if (clip == null) return;

        if (sfxPool.Count == 0) { AudioSource.PlayClipAtPoint(clip, position, volume); return; }

        var source = sfxPool.Dequeue();
        source.transform.position = position;
        source.clip = clip;
        source.volume = volume;
        source.spatialBlend = 1f;
        source.Play();
        StartCoroutine(ReturnSourceAfterPlay(source, clip.length));
    }

    public void PlayMusic(AudioClip clip, bool loop = true)
    {
        if (clip == null) return;
        musicSource.clip = clip;
        musicSource.loop = loop;
        musicSource.Play();
    }

    public void StopMusic() => musicSource.Stop();

    public void SetVolume(string parameter, float linear)
    {
        float db = linear > 0 ? Mathf.Log10(linear) * 20f : -80f;
        audioMixer.SetFloat(parameter, db);
        PlayerPrefs.SetFloat(parameter, linear);
    }

    private void LoadVolumeSettings()
    {
        SetVolume("MasterVolume", PlayerPrefs.GetFloat("MasterVolume", 1f));
        SetVolume("MusicVolume", PlayerPrefs.GetFloat("MusicVolume", 1f));
        SetVolume("SFXVolume", PlayerPrefs.GetFloat("SFXVolume", 1f));
    }

    private IEnumerator ReturnSourceAfterPlay(AudioSource source, float delay)
    {
        yield return new WaitForSeconds(delay);
        source.Stop();
        source.clip = null;
        sfxPool.Enqueue(source);
    }
}
```

**Audio Mixer hierarchy**: Master > Music, Master > SFX. Expose volume parameters as `MasterVolume`, `MusicVolume`, `SFXVolume`. Convert linear slider values to decibels with `Log10(value) * 20`.

## Networking (Netcode)

### Netcode for GameObjects Basics

```csharp
using Unity.Netcode;

public class NetworkPlayerController : NetworkBehaviour
{
    [SerializeField] private float moveSpeed = 5f;

    // Server-authoritative state synced to all clients
    private NetworkVariable<Vector3> networkPosition = new(
        writePerm: NetworkVariableWritePermission.Server);

    private NetworkVariable<int> networkHealth = new(
        writePerm: NetworkVariableWritePermission.Server);

    public override void OnNetworkSpawn()
    {
        if (IsOwner)
        {
            // Only the owning client handles input
            EnableInput();
        }

        networkHealth.OnValueChanged += OnHealthChanged;
    }

    private void Update()
    {
        if (!IsOwner) return;

        var input = new Vector2(Input.GetAxis("Horizontal"), Input.GetAxis("Vertical"));
        if (input.sqrMagnitude > 0.01f)
            MoveServerRpc(input);
    }

    // Client sends input to server
    [ServerRpc]
    private void MoveServerRpc(Vector2 input)
    {
        var move = new Vector3(input.x, 0, input.y) * moveSpeed * Time.deltaTime;
        transform.position += move;
        networkPosition.Value = transform.position;
    }

    // Server broadcasts damage to all clients
    [ClientRpc]
    private void TakeDamageClientRpc(int damage)
    {
        // Visual effects, sounds (run on all clients)
        PlayHitEffect();
    }

    private void OnHealthChanged(int oldVal, int newVal)
    {
        // Update UI on all clients
        UpdateHealthBar(newVal);
    }
}
```

**Networking rules**: Server-authoritative movement (client sends input, server validates and applies). Use `NetworkVariable` for state synchronization. Use `[ServerRpc]` for client-to-server calls. Use `[ClientRpc]` for server-to-all-clients calls. Check `IsOwner` before handling local input. Check `IsServer` before authoritative logic.

## Performance Optimization

### Profiling Checklist

1. **Unity Profiler** (`Window > Analysis > Profiler`): CPU, GPU, Memory, Rendering
2. **Frame Debugger** (`Window > Analysis > Frame Debugger`): Draw call analysis
3. **Memory Profiler** (package): Heap snapshots, leak detection
4. **Physics Debugger** (`Window > Analysis > Physics Debugger`): Collision visualization

### GC Allocation Avoidance

```csharp
// BAD: Allocating every frame
void Update()
{
    var enemies = FindObjectsOfType<Enemy>(); // GC alloc every frame
    var message = "Health: " + health;        // String concat = GC alloc
    var results = Physics.RaycastAll(ray);    // Allocates new array
}

// GOOD: Zero-allocation patterns
private readonly List<Enemy> enemyCache = new();
private readonly RaycastHit[] hitBuffer = new RaycastHit[10];
private readonly StringBuilder sb = new();

void Update()
{
    // Use NonAlloc physics methods
    int hitCount = Physics.RaycastNonAlloc(ray, hitBuffer);
    for (int i = 0; i < hitCount; i++) ProcessHit(hitBuffer[i]);

    // Use StringBuilder for strings
    sb.Clear();
    sb.Append("Health: ").Append(health);
    healthText.SetText(sb);

    // Use OverlapSphereNonAlloc instead of OverlapSphere
    int count = Physics.OverlapSphereNonAlloc(pos, radius, colliderBuffer, layerMask);
}
```

### Draw Call Optimization

```csharp
// 1. Static Batching: Mark non-moving objects as Static in Inspector
// 2. GPU Instancing: Enable on materials for repeated meshes
// 3. Sprite Atlases: Combine sprites to reduce texture switches

[CreateAssetMenu(menuName = "Game/Settings/Quality Settings")]
public class QualityConfig : ScriptableObject
{
    [Header("LOD")]
    public float[] lodDistances = { 15f, 30f, 60f };

    [Header("Rendering")]
    public int targetFrameRate = 60;
    public bool enableVSync = false;

    [Header("Physics")]
    public float fixedTimestep = 0.02f; // 50 Hz
    public int maxPhysicsIterations = 6;
}
```

### Shader and Material Guidelines

- Use SRP Batcher-compatible shaders (URP/HDRP Lit shader)
- Share materials across objects whenever possible (reduces draw calls)
- Use material property blocks for per-instance variations instead of new materials
- Avoid transparent materials on large surfaces (overdraw cost)

## Editor Scripting

### Custom Inspector

```csharp
#if UNITY_EDITOR
using UnityEditor;

[CustomEditor(typeof(EnemyConfig))]
public class EnemyConfigEditor : Editor
{
    public override void OnInspectorGUI()
    {
        DrawDefaultInspector();

        var config = (EnemyConfig)target;

        EditorGUILayout.Space();
        EditorGUILayout.LabelField("Calculated Values", EditorStyles.boldLabel);
        EditorGUILayout.LabelField("DPS", (config.damage / config.attackCooldown).ToString("F1"));
        EditorGUILayout.LabelField("TTK (100HP)",
            (100f / (config.damage / config.attackCooldown)).ToString("F1") + "s");

        if (GUILayout.Button("Reset to Defaults"))
        {
            Undo.RecordObject(config, "Reset Enemy Config");
            config.maxHealth = 50;
            config.damage = 10;
            config.moveSpeed = 3f;
            EditorUtility.SetDirty(config);
        }
    }
}
#endif
```

### Custom Build Script

```csharp
#if UNITY_EDITOR
using UnityEditor;
using UnityEditor.Build.Reporting;

public static class BuildScript
{
    private static readonly string[] Scenes =
    {
        "Assets/_Project/Scenes/MainMenu.unity",
        "Assets/_Project/Scenes/GameLevel.unity"
    };

    [MenuItem("Build/Windows")]
    public static void BuildWindows()
    {
        var report = BuildPipeline.BuildPlayer(
            Scenes, "Builds/Windows/MyGame.exe",
            BuildTarget.StandaloneWindows64, BuildOptions.None);

        LogBuildResult(report);
    }

    [MenuItem("Build/WebGL")]
    public static void BuildWebGL()
    {
        var report = BuildPipeline.BuildPlayer(
            Scenes, "Builds/WebGL",
            BuildTarget.WebGL, BuildOptions.None);

        LogBuildResult(report);
    }

    private static void LogBuildResult(BuildReport report)
    {
        if (report.summary.result == BuildResult.Succeeded)
            Debug.Log($"Build succeeded: {report.summary.totalSize} bytes");
        else
            Debug.LogError($"Build failed: {report.summary.result}");
    }
}
#endif
```

### Menu Items and Gizmos

```csharp
#if UNITY_EDITOR
public static class EditorTools
{
    [MenuItem("Tools/Clear PlayerPrefs")]
    public static void ClearPrefs()
    {
        PlayerPrefs.DeleteAll();
        Debug.Log("PlayerPrefs cleared.");
    }

    [MenuItem("Tools/Screenshot")]
    public static void CaptureScreenshot()
    {
        var filename = $"Screenshot_{System.DateTime.Now:yyyy-MM-dd_HH-mm-ss}.png";
        ScreenCapture.CaptureScreenshot(filename);
        Debug.Log($"Screenshot saved: {filename}");
    }
}
#endif

// Gizmos for debugging in Scene view
public class SpawnZone : MonoBehaviour
{
    [SerializeField] private float radius = 5f;
    [SerializeField] private Color gizmoColor = Color.yellow;

    private void OnDrawGizmosSelected()
    {
        Gizmos.color = gizmoColor;
        Gizmos.DrawWireSphere(transform.position, radius);
    }
}
```

## Testing Patterns

### Testing ScriptableObjects

```csharp
[TestFixture]
public class ItemDataTests
{
    [Test]
    public void GetRarityColor_ReturnsCorrectColor_ForEachRarity()
    {
        var item = ScriptableObject.CreateInstance<ItemData>();

        item.rarity = Rarity.Common;
        Assert.AreEqual(Color.white, item.GetRarityColor());

        item.rarity = Rarity.Legendary;
        Assert.AreEqual(new Color(1f, 0.5f, 0f), item.GetRarityColor());

        Object.DestroyImmediate(item); // Clean up ScriptableObject
    }
}
```

### Testing with Mocks (Service Locator)

```csharp
[TestFixture]
public class GameManagerTests
{
    private MockAudioManager mockAudio;

    [SetUp]
    public void SetUp()
    {
        mockAudio = new MockAudioManager();
        ServiceLocator.Register<IAudioManager>(mockAudio);
    }

    [TearDown]
    public void TearDown() => ServiceLocator.Clear();

    [Test]
    public void SetState_ToGameOver_StopsMusic()
    {
        var go = new GameObject();
        var manager = go.AddComponent<GameManager>();

        manager.SetState(GameState.GameOver);

        Assert.IsTrue(mockAudio.MusicStopped);
        Object.DestroyImmediate(go);
    }
}

// Simple mock (no external mocking framework needed)
public class MockAudioManager : IAudioManager
{
    public bool MusicStopped { get; private set; }

    public void PlaySFX(AudioClip clip, float volume = 1f) { }
    public void PlaySFXAtPosition(AudioClip clip, Vector3 pos, float vol = 1f) { }
    public void PlayMusic(AudioClip clip, bool loop = true) { }
    public void StopMusic() => MusicStopped = true;
    public void SetVolume(string parameter, float volume) { }
}
```

### Play Mode Test with Coroutine

```csharp
[TestFixture]
public class PlayerHealthPlayTests
{
    private GameObject playerObject;
    private Health playerHealth;

    [SetUp]
    public void SetUp()
    {
        playerObject = new GameObject("TestPlayer");
        playerHealth = playerObject.AddComponent<Health>();
    }

    [TearDown]
    public void TearDown() => Object.Destroy(playerObject);

    [UnityTest]
    public IEnumerator TakeDamage_AfterStart_ReducesHealth()
    {
        yield return null; // Allow Start() to execute

        playerHealth.TakeDamage(30);

        Assert.AreEqual(70, playerHealth.Current);
        Assert.IsTrue(playerHealth.IsAlive);
    }

    [UnityTest]
    public IEnumerator TakeLethalDamage_TriggersDeath()
    {
        yield return null;

        bool deathTriggered = false;
        playerHealth.OnDeath.AddListener(() => deathTriggered = true);

        playerHealth.TakeDamage(999);

        Assert.IsTrue(deathTriggered);
        Assert.IsFalse(playerHealth.IsAlive);
        Assert.AreEqual(0, playerHealth.Current);
    }
}
```

## Advanced Object Pooling

### IPoolable Interface

```csharp
public interface IPoolable
{
    void OnSpawn();  // Called when retrieved from pool
    void OnDespawn(); // Called when returned to pool
}

public class PooledBullet : MonoBehaviour, IPoolable
{
    [SerializeField] private float lifetime = 3f;
    [SerializeField] private float speed = 20f;

    private float timer;

    public void OnSpawn()
    {
        timer = lifetime;
    }

    public void OnDespawn()
    {
        // Reset state
    }

    private void Update()
    {
        transform.Translate(Vector3.forward * speed * Time.deltaTime);

        timer -= Time.deltaTime;
        if (timer <= 0f)
            BulletPool.Instance.Return(this);
    }

    private void OnTriggerEnter(Collider other)
    {
        if (other.TryGetComponent<IDamageable>(out var target))
            target.TakeDamage(10);

        BulletPool.Instance.Return(this);
    }
}
```

### Unity 2021+ Built-in Pool

```csharp
using UnityEngine.Pool;

public class ParticleSpawner : MonoBehaviour
{
    [SerializeField] private ParticleSystem prefab;

    private ObjectPool<ParticleSystem> pool;

    private void Awake()
    {
        pool = new ObjectPool<ParticleSystem>(
            createFunc: () => Instantiate(prefab, transform),
            actionOnGet: ps => { ps.gameObject.SetActive(true); ps.Play(); },
            actionOnRelease: ps => { ps.Stop(); ps.gameObject.SetActive(false); },
            actionOnDestroy: ps => Destroy(ps.gameObject),
            defaultCapacity: 10,
            maxSize: 50
        );
    }

    public void SpawnEffect(Vector3 position)
    {
        var ps = pool.Get();
        ps.transform.position = position;
        StartCoroutine(ReturnAfterDuration(ps, ps.main.duration));
    }

    private IEnumerator ReturnAfterDuration(ParticleSystem ps, float duration)
    {
        yield return new WaitForSeconds(duration);
        pool.Release(ps);
    }
}
```

## Coroutine Patterns

### Sequenced Actions

```csharp
public class CutsceneController : MonoBehaviour
{
    public IEnumerator PlayIntro()
    {
        yield return FadeIn(1f);
        yield return ShowDialogue("Welcome, hero.");
        yield return new WaitForSeconds(1f);
        yield return MoveCamera(targetPosition, 2f);
        yield return FadeOut(0.5f);

        // Continue to gameplay
        GameManager.Instance.SetState(GameState.Playing);
    }

    private IEnumerator FadeIn(float duration)
    {
        // Fade from black
        yield return TweenHelper.FadeCanvasGroup(fadeGroup, 0f, duration);
    }

    private IEnumerator ShowDialogue(string text)
    {
        dialogueText.text = text;
        dialoguePanel.SetActive(true);
        yield return new WaitUntil(() => Input.anyKeyDown);
        dialoguePanel.SetActive(false);
    }

    private IEnumerator MoveCamera(Vector3 target, float duration)
    {
        yield return TweenHelper.LerpPosition(Camera.main.transform, target, duration);
    }

    private IEnumerator FadeOut(float duration)
    {
        yield return TweenHelper.FadeCanvasGroup(fadeGroup, 1f, duration);
    }
}
```

### Cancellable Coroutine Pattern

```csharp
public class AbilitySystem : MonoBehaviour
{
    private Coroutine activeAbility;

    public void UseAbility(IAbility ability)
    {
        // Cancel current ability if active
        if (activeAbility != null)
            StopCoroutine(activeAbility);

        activeAbility = StartCoroutine(ExecuteAbility(ability));
    }

    private IEnumerator ExecuteAbility(IAbility ability)
    {
        ability.OnStart(gameObject);

        yield return new WaitForSeconds(ability.CastTime);

        ability.OnExecute(gameObject);

        yield return new WaitForSeconds(ability.Duration);

        ability.OnEnd(gameObject);
        activeAbility = null;
    }
}

public interface IAbility
{
    float CastTime { get; }
    float Duration { get; }
    void OnStart(GameObject caster);
    void OnExecute(GameObject caster);
    void OnEnd(GameObject caster);
}
```

### WaitForSeconds Cache

```csharp
// Avoid allocating new WaitForSeconds every time
public static class WaitCache
{
    private static readonly Dictionary<float, WaitForSeconds> Cache = new();

    public static WaitForSeconds Seconds(float duration)
    {
        if (!Cache.TryGetValue(duration, out var wait))
        {
            wait = new WaitForSeconds(duration);
            Cache[duration] = wait;
        }
        return wait;
    }

    // Usage: yield return WaitCache.Seconds(0.5f);
}
```
