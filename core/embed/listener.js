() =>
  new Promise((resolve) => {
    let cap;
    const done = () => {
      clearTimeout(cap);
      resolve();
    };
    cap = setTimeout(done, 1000);
    if (typeof requestIdleCallback === "function") {
      requestIdleCallback(done, { timeout: 250 });
    } else {
      setTimeout(done, 250);
    }
  });
