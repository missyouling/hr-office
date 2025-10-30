#!/usr/bin/env node

/**
 * Supabase 连接测试脚本
 * 用法: cd frontend && node ../scripts/test-supabase-connection.js
 * 或者: node scripts/test-supabase-connection.js（从项目根目录）
 */

const fs = require('fs');
const path = require('path');

// 动态加载 @supabase/supabase-js（需要从 frontend 目录加载）
let createClient;
const frontendPath = path.join(__dirname, '..', 'frontend');
const modulePath = path.join(frontendPath, 'node_modules', '@supabase', 'supabase-js');

try {
  const supabaseModule = require(modulePath);
  createClient = supabaseModule.createClient;
} catch (err) {
  console.error('❌ 无法加载 @supabase/supabase-js 模块');
  console.error('   请确保已在 frontend 目录运行 npm install');
  console.error(`   模块路径: ${modulePath}`);
  process.exit(1);
}

// 颜色输出
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

// 读取环境变量
function loadEnv() {
  const envPath = path.join(__dirname, '..', 'frontend', '.env.local');
  
  if (!fs.existsSync(envPath)) {
    log('❌ 错误: frontend/.env.local 文件不存在', 'red');
    process.exit(1);
  }

  const envContent = fs.readFileSync(envPath, 'utf8');
  const envVars = {};
  
  envContent.split('\n').forEach(line => {
    const match = line.match(/^([^=:#]+)=(.*)$/);
    if (match) {
      const key = match[1].trim();
      const value = match[2].trim();
      envVars[key] = value;
    }
  });

  return envVars;
}

async function testConnection() {
  log('\n🔧 Supabase 连接测试工具', 'blue');
  log('================================\n', 'blue');

  // 1. 加载环境变量
  log('📋 步骤 1: 加载环境变量...', 'yellow');
  const env = loadEnv();
  
  const supabaseUrl = env.NEXT_PUBLIC_SUPABASE_URL;
  const supabaseKey = env.NEXT_PUBLIC_SUPABASE_ANON_KEY;

  if (!supabaseUrl || !supabaseKey) {
    log('❌ 错误: 环境变量配置不完整', 'red');
    log('   请检查 frontend/.env.local 文件', 'red');
    process.exit(1);
  }

  log(`✅ URL: ${supabaseUrl}`, 'green');
  log(`✅ Key: ${supabaseKey.substring(0, 20)}...`, 'green');

  // 2. 创建客户端
  log('\n📋 步骤 2: 创建 Supabase 客户端...', 'yellow');
  const supabase = createClient(supabaseUrl, supabaseKey);
  log('✅ 客户端创建成功', 'green');

  // 3. 测试数据库连接
  log('\n📋 步骤 3: 测试数据库连接...', 'yellow');
  try {
    const { data, error } = await supabase
      .from('profiles')
      .select('count')
      .limit(1);

    if (error) {
      log(`❌ 数据库连接失败: ${error.message}`, 'red');
      log('   提示: 请检查表是否已创建', 'yellow');
      process.exit(1);
    }

    log('✅ 数据库连接成功', 'green');
  } catch (err) {
    log(`❌ 连接异常: ${err.message}`, 'red');
    process.exit(1);
  }

  // 4. 检查表结构
  log('\n📋 步骤 4: 检查表结构...', 'yellow');
  const tables = [
    'profiles',
    'periods',
    'source_files',
    'raw_records',
    'period_summaries',
    'personal_charges',
    'unit_charges',
    'roster_entries',
    'audit_logs'
  ];

  let allTablesExist = true;
  for (const table of tables) {
    try {
      const { data, error } = await supabase
        .from(table)
        .select('count')
        .limit(1);

      if (error && error.code === 'PGRST116') {
        log(`❌ 表不存在: ${table}`, 'red');
        allTablesExist = false;
      } else {
        log(`✅ 表存在: ${table}`, 'green');
      }
    } catch (err) {
      log(`❌ 检查表失败: ${table} - ${err.message}`, 'red');
      allTablesExist = false;
    }
  }

  if (!allTablesExist) {
    log('\n⚠️  警告: 部分表不存在，请执行数据库迁移 SQL', 'yellow');
  }

  // 5. 测试认证功能
  log('\n📋 步骤 5: 测试认证功能...', 'yellow');
  try {
    const { data: { session }, error } = await supabase.auth.getSession();
    
    if (error) {
      log(`⚠️  认证检查: ${error.message}`, 'yellow');
    } else {
      log('✅ 认证系统正常（当前未登录）', 'green');
    }
  } catch (err) {
    log(`❌ 认证测试失败: ${err.message}`, 'red');
  }

  // 总结
  log('\n================================', 'blue');
  log('🎉 连接测试完成！\n', 'blue');
  
  if (allTablesExist) {
    log('✅ 所有检查项通过', 'green');
    log('\n下一步:', 'yellow');
    log('1. 启动前端开发服务器: cd frontend && npm run dev', 'yellow');
    log('2. 启动后端开发服务器: cd backend && go run .', 'yellow');
  } else {
    log('⚠️  存在问题，请按照提示修复', 'yellow');
  }
}

// 执行测试
testConnection().catch(err => {
  log(`\n❌ 测试失败: ${err.message}`, 'red');
  console.error(err);
  process.exit(1);
});