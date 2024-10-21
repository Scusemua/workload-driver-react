global.__globalCustomDomain = function () {
    const environment = process.env.NODE_ENV || 'development';
    if (environment.trim().toLowerCase() === 'production') {
        return process.env.PUBLIC_PATH || '/AAAAAAAAA';
    }

    return '/AAAAAAAAAAA';
};
