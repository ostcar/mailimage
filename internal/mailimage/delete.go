package mailimage

import "golang.org/x/xerrors"

func Delete(id int) error {
	pool, err := newPool(redisAddr)
	if err != nil {
		return xerrors.Errorf("Can not create redis pool: %w", err)
	}

	if err = pool.deleteFromID(id); err != nil {
		return xerrors.Errorf("Can not delete image: %w", err)
	}
	return nil
}
