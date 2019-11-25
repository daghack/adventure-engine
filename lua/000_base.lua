scalars = {}

function printR(a)
	print(a)
	return a
end

function sum(a, b)
	return a + b
end

function if_else(cond, a, b)
	if cond then
		return a
	else
		return b
	end
end

function get_scalar(key, default)
	if not scalars[key] and default then
		scalars[key] = default
	end
	return scalars[key]
end

function new_table()
	return {}
end

function table_index(scalar_key, key, default)
	if not scalars[scalar_key][key] and default then
		scalars[scalar_key][key] = default
	end
	return scalars[scalar_key][key]
end
