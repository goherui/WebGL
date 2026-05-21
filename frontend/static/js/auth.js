// 认证状态检查
async function checkAuth() {
    try {
        const response = await fetch('/api/check-auth', {
            method: 'GET',
            credentials: 'include'
        });
        const data = await response.json();
        
        if (data.code === 0 && data.data) {
            return data.data;
        }
        return null;
    } catch (error) {
        console.error('检查认证失败:', error);
        return null;
    }
}

// 退出登录
async function handleLogout() {
    try {
        const response = await fetch('/api/logout', {
            method: 'POST',
            credentials: 'include'
        });
        
        // 无论成功失败都重定向到登录页
        window.location.href = '/';
    } catch (error) {
        console.error('退出失败:', error);
        // 仍然重定向
        window.location.href = '/';
    }
}

// Toast提示
function showToast(message, type = 'success') {
    // 移除已存在的toast
    const existingToast = document.querySelector('.toast');
    if (existingToast) {
        existingToast.remove();
    }

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    document.body.appendChild(toast);

    // 触发动画
    requestAnimationFrame(() => {
        toast.classList.add('show');
    });

    // 3秒后移除
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 400);
    }, 3000);
}

// 导出函数供全局使用
window.checkAuth = checkAuth;
window.handleLogout = handleLogout;
window.showToast = showToast;
