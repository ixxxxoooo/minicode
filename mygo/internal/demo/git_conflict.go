package demo

// 合并冲突演示文件正文(含冲突标记).

const conflictTwo = `fn load() -> &'static str {
<<<<<<< HEAD
    "local-cache"
=======
    "redis-cluster"
>>>>>>> origin/feat/cache
}

fn ttl() -> u64 {
    3600
}
`

const conflictFive = `[package]
name = "router-demo"

[dependencies]
<<<<<<< HEAD
axum = "0.8"
tokio = { version = "1", features = ["full"] }
||||||| merged common ancestors
axum = "0.7"
=======
axum = { version = "0.8", features = ["macros"] }
serde = "1"
>>>>>>> feat/serde
hyper = "1"

[features]
<<<<<<< HEAD
default = ["trace"]
=======
default = []
extra = ["serde"]
>>>>>>> feat/serde
`

const conflictBroken = `# 说明

<<<<<<< HEAD
这一段故意没有闭合标记
保留损坏文件用于界面提示

普通段落仍应显示
`

const conflictBothAdded = `{
<<<<<<< HEAD
  "timeout": 30
=======
  "timeout": 60,
  "retries": 3
>>>>>>> incoming
}
`

const conflictDeleteModify = `<<<<<<< HEAD
已删除:传入侧仍保留三行
=======
line one
line two
line three
>>>>>>> branch
`
