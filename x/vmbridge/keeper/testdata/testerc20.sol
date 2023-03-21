// SPDX-License-Identifier: MIT

pragma solidity 0.8.7;

contract Exchange is ERC20 {
    address public constant moduleAddress =
    address(0xc63cf6c8E1f3DF41085E9d8Af49584dae1432b4f);

    string public wasmContractAddress = "ex14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s6fqu27";

    event __OKBCSendToWasm(string wasmAddr, string recipient, uint256 amount);

    function initialize(string memory denom_, uint8 decimals_) public {
        __ERC20_init(denom_, denom_, decimals_);
    }

    function native_denom() public view returns (string memory) {
        return symbol();
    }

    function updatewasmContractAddress(string memory addr) public  {
        wasmContractAddress = addr;
    }

    function mint(address recipient,uint256 amount) public {
        _mint(recipient, amount);
    }


    function mintERC20(string calldata caller, address recipient,uint256 amount) public returns (bool) {
        require(msg.sender == moduleAddress);
        require(keccak256(abi.encodePacked(caller)) == keccak256(abi.encodePacked(wasmContractAddress)));
        _mint(recipient, amount);
        return true;
    }


    // send an "amount" of the contract token to recipient on wasm
    function send_to_wasm(string memory recipient,string memory wasmContract , uint256 amount) public {
        _burn(msg.sender, amount);
        emit __OKBCSendToWasm(wasmContract,recipient, amount);
    }
}

contract ERC20 {
    bool private initialized;

    string private _name;
    string private _symbol;
    uint8 private _decimals;
    uint256 private _totalSupply;

    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(
        address indexed owner,
        address indexed spender,
        uint256 value
    );

    function __ERC20_init(
        string memory name_,
        string memory symbol_,
        uint8 decimals_
    ) internal {
        require(!initialized, "ERC20: already initialized;");
        initialized = true;

        _name = name_;
        _symbol = symbol_;
        _decimals = decimals_;
    }

    function name() public view virtual returns (string memory) {
        return _name;
    }

    function symbol() public view virtual returns (string memory) {
        return _symbol;
    }

    function decimals() public view virtual returns (uint8) {
        return _decimals;
    }

    function totalSupply() public view virtual returns (uint256) {
        return _totalSupply;
    }

    function balanceOf(address account) public view virtual returns (uint256) {
        return _balances[account];
    }

    function transfer(address to, uint256 amount)
    public
    virtual
    returns (bool)
    {
        address owner = msg.sender;
        _transfer(owner, to, amount);
        return true;
    }

    function allowance(address owner, address spender)
    public
    view
    virtual
    returns (uint256)
    {
        return _allowances[owner][spender];
    }

    function approve(address spender, uint256 amount)
    public
    virtual
    returns (bool)
    {
        address owner = msg.sender;
        _approve(owner, spender, amount);
        return true;
    }

    function transferFrom(
        address from,
        address to,
        uint256 amount
    ) public virtual returns (bool) {
        address spender = msg.sender;
        _spendAllowance(from, spender, amount);
        _transfer(from, to, amount);
        return true;
    }

    function increaseAllowance(address spender, uint256 addedValue)
    public
    virtual
    returns (bool)
    {
        address owner = msg.sender;
        _approve(owner, spender, _allowances[owner][spender] + addedValue);
        return true;
    }

    function decreaseAllowance(address spender, uint256 subtractedValue)
    public
    virtual
    returns (bool)
    {
        address owner = msg.sender;
        uint256 currentAllowance = _allowances[owner][spender];
        require(
            currentAllowance >= subtractedValue,
            "ERC20: decreased allowance below zero"
        );
    unchecked {
        _approve(owner, spender, currentAllowance - subtractedValue);
    }

        return true;
    }

    function _transfer(
        address from,
        address to,
        uint256 amount
    ) internal virtual {
        require(from != address(0), "ERC20: transfer from the zero address");
        require(to != address(0), "ERC20: transfer to the zero address");

        uint256 fromBalance = _balances[from];
        require(
            fromBalance >= amount,
            "ERC20: transfer amount exceeds balance"
        );
    unchecked {
        _balances[from] = fromBalance - amount;
    }
        _balances[to] += amount;

        emit Transfer(from, to, amount);
    }

    function _mint(address account, uint256 amount) internal virtual {
        require(account != address(0), "ERC20: mint to the zero address");

        _totalSupply += amount;
        _balances[account] += amount;
        emit Transfer(address(0), account, amount);
    }

    function _burn(address account, uint256 amount) internal virtual {
        require(account != address(0), "ERC20: burn from the zero address");

        uint256 accountBalance = _balances[account];
        require(accountBalance >= amount, "ERC20: burn amount exceeds balance");
    unchecked {
        _balances[account] = accountBalance - amount;
    }
        _totalSupply -= amount;

        emit Transfer(account, address(0), amount);
    }

    function _approve(
        address owner,
        address spender,
        uint256 amount
    ) internal virtual {
        require(owner != address(0), "ERC20: approve from the zero address");
        require(spender != address(0), "ERC20: approve to the zero address");

        _allowances[owner][spender] = amount;
        emit Approval(owner, spender, amount);
    }

    function _spendAllowance(
        address owner,
        address spender,
        uint256 amount
    ) internal virtual {
        uint256 currentAllowance = allowance(owner, spender);
        if (currentAllowance != type(uint256).max) {
            require(
                currentAllowance >= amount,
                "ERC20: insufficient allowance"
            );
        unchecked {
            _approve(owner, spender, currentAllowance - amount);
        }
        }
    }
}
